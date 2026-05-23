mod config;
mod cost;
mod organization;
mod sts;

use std::io::Write;
use std::process;

use anyhow::Result;
use aws_sdk_costexplorer::Client as CeClient;
use aws_sdk_iam::Client as IamClient;
use aws_sdk_organizations::Client as OrganizationsClient;
use aws_sdk_sts::Client as StsClient;
use chrono::NaiveDate;
use clap::Parser;
use comfy_table::{Attribute, Cell, CellAlignment, ContentArrangement, Table};
use dialoguer::MultiSelect;
use serde::Serialize;

use cost::{Cost, GetCostsOption};
use organization::{Account, Accounts};

const ERR_AWS_ORGANIZATION_NOT_ENABLED: &str =
    "This AWS account is not part of AWS Organizations organization. ";
const ERR_INSUFFICIENT_IAM_PERMISSIONS: &str =
    "Failed to perform \"{}\".  Make sure you have sufficient IAM permissions. See https://github.com/toricls/acos#prerequisites for the details.";

/// An interactive CLI tool to retrieve and show your AWS costs.
#[derive(Parser, Debug)]
#[command(name = "acos", version, about)]
struct Cli {
    /// The ID of an AWS Organizational Unit (OU) or Root to list direct-children AWS accounts.
    /// It must start with 'ou-' or 'r-' prefix. This flag is ignored when --account-ids is set.
    #[arg(long = "ou", default_value = "")]
    ou: String,

    /// The date to retrieve the cost data. The format should be 'YYYY-MM-DD'.
    /// The default value is today in UTC.
    #[arg(long = "as-of", default_value = "")]
    as_of: String,

    /// The cost of this month will be compared to either one of 'YESTERDAY' or 'LAST_WEEK'.
    /// This flag is ignored when --json is set.
    #[arg(long = "compared-to", default_value = "YESTERDAY")]
    compared_to: String,

    /// Print JSON instead of table.
    #[arg(long = "json", default_value_t = false)]
    json: bool,

    /// Comma-separated AWS account IDs to retrieve costs.
    /// The interactive account selector is skipped when this flag is set.
    #[arg(long = "account-ids", default_value = "")]
    account_ids: String,
}

#[tokio::main]
async fn main() {
    if let Err(e) = run().await {
        eprintln!("{}", e);
        process::exit(1);
    }
}

async fn run() -> Result<()> {
    let cli = Cli::parse();

    // Parse as_of date
    let as_of = if cli.as_of.is_empty() {
        chrono::Utc::now().date_naive()
    } else {
        match NaiveDate::parse_from_str(&cli.as_of, "%Y-%m-%d") {
            Ok(d) => d,
            Err(_) => {
                eprintln!(
                    "error invalid date format for the --as-of flag. It should be 'YYYY-MM-DD'."
                );
                process::exit(1);
            }
        }
    };

    // Validate compared_to
    match cli.compared_to.as_str() {
        "YESTERDAY" | "LAST_WEEK" => {}
        _ => {
            eprintln!("error invalid value for the --compared-to flag. It should be either 'YESTERDAY' or 'LAST_WEEK'.");
            process::exit(2);
        }
    }

    // Parse account IDs
    let account_ids: Vec<String> = if cli.account_ids.is_empty() {
        vec![]
    } else {
        cli.account_ids
            .split(',')
            .map(|s| s.trim().to_string())
            .filter(|s| !s.is_empty())
            .collect()
    };

    // Load AWS config and create clients
    let aws_cfg = config::load_aws_config().await;
    let org_client = OrganizationsClient::new(&aws_cfg);
    let sts_client = StsClient::new(&aws_cfg);
    let iam_client = IamClient::new(&aws_cfg);
    let ce_client = CeClient::new(&aws_cfg);

    // Get candidate accounts
    let candidate_accounts = get_accounts(
        &org_client,
        &sts_client,
        &iam_client,
        &account_ids,
        &cli.ou,
    )
    .await?;

    // Select accounts
    let selected_accounts = if !account_ids.is_empty() {
        // Skip interactive selector when account IDs are explicitly provided
        candidate_accounts
    } else {
        prompt_accounts_selection(&candidate_accounts)?
    };

    // Get costs
    let opt = GetCostsOption::new(as_of);
    let costs = cost::get_costs(&ce_client, &selected_accounts, &opt).await?;

    // Sort by account ID
    let mut cost_array: Vec<Cost> = costs.into_values().collect();
    cost_array.sort_by(|a, b| a.account_id.cmp(&b.account_id));

    if cli.json {
        print_json(&cost_array, &as_of)?;
    } else {
        print_table(&cost_array, &cli.compared_to, &as_of);
    }

    Ok(())
}

async fn get_accounts(
    org_client: &OrganizationsClient,
    sts_client: &StsClient,
    iam_client: &IamClient,
    account_ids: &[String],
    ou_id: &str,
) -> Result<Accounts> {
    let result = if !account_ids.is_empty() {
        eprintln!("Account IDs specified. Retrieving accounts information...");
        get_accounts_by_ids(org_client, account_ids).await
    } else if !ou_id.is_empty() {
        eprintln!(
            "Retrieving AWS accounts under the OU '{}'...",
            ou_id
        );
        let r = organization::list_accounts_by_ou(org_client, ou_id).await;
        if let Err(ref e) = r {
            if organization::ou_not_found(e) {
                anyhow::bail!("error the OU \"{}\" doesn't exist", ou_id);
            }
        }
        r
    } else {
        organization::list_accounts(org_client).await
    };

    match result {
        Ok(accounts) => Ok(accounts),
        Err(ref e) => {
            if organization::is_organization_not_enabled(e) {
                eprint!("{}", ERR_AWS_ORGANIZATION_NOT_ENABLED);
            } else if organization::has_no_permission_to_organizations_api(e) {
                eprintln!(
                    "{}",
                    ERR_INSUFFICIENT_IAM_PERMISSIONS.replace("{}", "organizations:ListAccounts")
                );
            }
            eprintln!("Falling back to using \"sts:GetCallerIdentity\" and \"iam:ListAccountAliases\" to obtain your AWS account information... ");
            get_caller_account(sts_client, iam_client).await
        }
    }
}

async fn get_accounts_by_ids(
    org_client: &OrganizationsClient,
    account_ids: &[String],
) -> Result<Accounts> {
    let all_accounts = organization::list_accounts(org_client).await?;
    let mut result = Accounts::new();

    for id in account_ids {
        if id.is_empty() {
            continue;
        }
        if let Some(acc) = all_accounts.get(id) {
            result.insert(id.clone(), acc.clone());
        } else {
            eprintln!("Account ID '{}' is not found in your AWS organization", id);
        }
    }

    Ok(result)
}

async fn get_caller_account(sts_client: &StsClient, iam_client: &IamClient) -> Result<Accounts> {
    let (account_id, account_name) = sts::get_caller_account(sts_client, iam_client).await?;
    let mut accounts = Accounts::new();
    accounts.insert(
        account_id.clone(),
        Account {
            id: account_id,
            name: account_name,
        },
    );
    Ok(accounts)
}

fn prompt_accounts_selection(accounts: &Accounts) -> Result<Accounts> {
    if accounts.is_empty() {
        anyhow::bail!("error no accounts found");
    }

    if accounts.len() == 1 {
        return Ok(accounts.clone());
    }

    // Sort accounts for consistent ordering
    let mut sorted_accounts: Vec<(&String, &Account)> = accounts.iter().collect();
    sorted_accounts.sort_by_key(|(id, _)| id.to_string());

    let options: Vec<String> = sorted_accounts
        .iter()
        .map(|(_, acc)| format!("{} - {}", acc.id, acc.name))
        .collect();

    let selections = MultiSelect::new()
        .with_prompt("Select accounts")
        .items(&options)
        .interact_on(&dialoguer::console::Term::stderr())?;

    if selections.is_empty() {
        anyhow::bail!("error no accounts selected");
    }

    let mut result = Accounts::new();
    for idx in selections {
        let (id, acc) = &sorted_accounts[idx];
        result.insert(id.to_string(), (*acc).clone());
    }

    Ok(result)
}

#[derive(Serialize)]
struct JsonOutput {
    #[serde(rename = "AsOf")]
    as_of: String,
    #[serde(rename = "Costs")]
    costs: Vec<Cost>,
}

fn print_json(costs: &[Cost], as_of: &NaiveDate) -> Result<()> {
    let output = JsonOutput {
        as_of: format!("{}T00:00:00Z", as_of.format("%Y-%m-%d")),
        costs: costs.to_vec(),
    };
    let json_str = serde_json::to_string(&output)?;
    println!("{}", json_str);
    Ok(())
}

fn print_table(costs: &[Cost], compared_to: &str, as_of: &NaiveDate) {
    let incr_header = if compared_to == "LAST_WEEK" {
        "vs Last Week ($)"
    } else {
        "vs Yesterday ($)"
    };

    let mut table = Table::new();
    table.set_content_arrangement(ContentArrangement::Dynamic);
    table.set_header(vec![
        Cell::new("Account ID"),
        Cell::new("Account Name"),
        Cell::new("This Month ($)").set_alignment(CellAlignment::Right),
        Cell::new(incr_header).set_alignment(CellAlignment::Right),
        Cell::new("Last Month ($)").set_alignment(CellAlignment::Right),
    ]);

    let mut total_this_month = 0.0_f64;
    let mut total_increase = 0.0_f64;
    let mut total_last_month = 0.0_f64;

    for c in costs {
        let incr = if compared_to == "LAST_WEEK" {
            c.latest_weekly_cost_increase
        } else {
            c.latest_daily_cost_increase
        };

        let incr_str = format!("{} {:.6}", get_amount_prefix(incr), incr.abs());

        table.add_row(vec![
            Cell::new(&c.account_id),
            Cell::new(&c.account_name),
            Cell::new(format!("{:.6}", c.amount_this_month)).set_alignment(CellAlignment::Right),
            Cell::new(&incr_str).set_alignment(CellAlignment::Right),
            Cell::new(format!("{:.6}", c.amount_last_month)).set_alignment(CellAlignment::Right),
        ]);

        total_this_month += c.amount_this_month;
        total_increase += incr;
        total_last_month += c.amount_last_month;
    }

    // Footer row
    let total_incr_str = format!(
        "{} {:.6}",
        get_amount_prefix(total_increase),
        total_increase.abs()
    );
    table.add_row(vec![
        Cell::new("").add_attribute(Attribute::Bold),
        Cell::new("Total").add_attribute(Attribute::Bold),
        Cell::new(format!("{:.6}", total_this_month))
            .set_alignment(CellAlignment::Right)
            .add_attribute(Attribute::Bold),
        Cell::new(&total_incr_str)
            .set_alignment(CellAlignment::Right)
            .add_attribute(Attribute::Bold),
        Cell::new(format!("{:.6}", total_last_month))
            .set_alignment(CellAlignment::Right)
            .add_attribute(Attribute::Bold),
    ]);

    println!("{table}");
    println!("As of {}.", as_of.format("%Y-%m-%d"));

    // Flush stdout
    let _ = std::io::stdout().flush();
}

fn get_amount_prefix(amount: f64) -> &'static str {
    if amount > 0.0 {
        "+"
    } else if amount < 0.0 {
        "-"
    } else {
        ""
    }
}
