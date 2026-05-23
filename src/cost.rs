use std::collections::HashMap;

use anyhow::{bail, Result};
use aws_sdk_costexplorer::types::{
    DateInterval, DimensionValues, Expression, GroupDefinition, GroupDefinitionType,
};
use aws_sdk_costexplorer::Client as CeClient;
use chrono::{Datelike, NaiveDate};
use serde::Serialize;

use crate::organization::{Accounts, AccountsExt};

const CE_DATA_GRANULARITY: aws_sdk_costexplorer::types::Granularity =
    aws_sdk_costexplorer::types::Granularity::Daily;
const CE_COST_METRIC: &str = "UnblendedCost";
const CE_COST_GROUP_BY: &str = "LINKED_ACCOUNT";

/// Options for GetCosts.
#[derive(Debug, Clone)]
pub struct GetCostsOption {
    pub exclude_credit: bool,
    pub exclude_upfront: bool,
    pub exclude_refund: bool,
    pub exclude_support: bool,

    pub as_of: String,
    pub one_week_ago: String,
    pub first_day_of_last_month: String,
    pub first_day_of_this_month: String,
}

impl GetCostsOption {
    pub fn new(as_of_date: NaiveDate) -> Self {
        let one_week_ago = as_of_date - chrono::Duration::days(7);
        let first_day_of_this_month =
            NaiveDate::from_ymd_opt(as_of_date.year(), as_of_date.month(), 1).unwrap();

        let (prev_year, prev_month) = if as_of_date.month() == 1 {
            (as_of_date.year() - 1, 12)
        } else {
            (as_of_date.year(), as_of_date.month() - 1)
        };
        let first_day_of_last_month = NaiveDate::from_ymd_opt(prev_year, prev_month, 1).unwrap();

        let fmt = "%Y-%m-%d";
        Self {
            exclude_credit: true,
            exclude_upfront: true,
            exclude_refund: false,
            exclude_support: false,
            as_of: as_of_date.format(fmt).to_string(),
            one_week_ago: one_week_ago.format(fmt).to_string(),
            first_day_of_this_month: first_day_of_this_month.format(fmt).to_string(),
            first_day_of_last_month: first_day_of_last_month.format(fmt).to_string(),
        }
    }
}

/// Represents a cost for a given account.
#[derive(Debug, Clone, Serialize)]
pub struct Cost {
    #[serde(rename = "AccountID")]
    pub account_id: String,
    #[serde(rename = "AccountName")]
    pub account_name: String,
    #[serde(rename = "LatestDailyCostIncrease")]
    pub latest_daily_cost_increase: f64,
    #[serde(rename = "LatestWeeklyCostIncrease")]
    pub latest_weekly_cost_increase: f64,
    #[serde(rename = "AmountLastMonth")]
    pub amount_last_month: f64,
    #[serde(rename = "AmountThisMonth")]
    pub amount_this_month: f64,
}

/// A map of account ID -> Cost.
pub type Costs = HashMap<String, Cost>;

/// Get costs for the given accounts.
pub async fn get_costs(
    client: &CeClient,
    accounts: &Accounts,
    opt: &GetCostsOption,
) -> Result<Costs> {
    let account_ids = accounts.account_ids();
    if account_ids.is_empty() {
        bail!("error no account to retrieve cost: GetCosts requires at least one account in Accounts");
    }

    // Initialize costs map with all accounts (some may have no billing data yet)
    let mut costs: Costs = HashMap::new();
    for (id, acc) in accounts {
        costs.insert(
            id.clone(),
            Cost {
                account_id: acc.id.clone(),
                account_name: acc.name.clone(),
                latest_daily_cost_increase: 0.0,
                latest_weekly_cost_increase: 0.0,
                amount_last_month: 0.0,
                amount_this_month: 0.0,
            },
        );
    }

    let ce_input = build_cost_explorer_input(opt, &account_ids);

    let mut next_token: Option<String> = None;
    loop {
        let mut req = client
            .get_cost_and_usage()
            .granularity(CE_DATA_GRANULARITY)
            .set_metrics(Some(vec![CE_COST_METRIC.to_string()]))
            .time_period(ce_input.time_period.clone())
            .set_group_by(Some(ce_input.group_by.clone()))
            .set_filter(Some(ce_input.filter.clone()));

        if let Some(token) = &next_token {
            req = req.next_page_token(token);
        }

        let output = req.send().await?;

        let mut this_month = false;
        let mut last_week = false;

        for r in output.results_by_time() {
            let period = r.time_period().unwrap();
            let start = period.start();
            let end = period.end();

            if start == opt.first_day_of_this_month {
                this_month = true;
            }
            if this_month && start == opt.one_week_ago {
                last_week = true;
            }

            for g in r.groups() {
                let account_id = g.keys().first().map(|s| s.as_str()).unwrap_or_default();
                let amount: f64 = g
                    .metrics()
                    .and_then(|m| m.get(CE_COST_METRIC))
                    .and_then(|v| v.amount())
                    .and_then(|a| a.parse::<f64>().ok())
                    .unwrap_or(0.0);

                if let Some(c) = costs.get_mut(account_id) {
                    if this_month {
                        c.amount_this_month += amount;
                    } else {
                        c.amount_last_month += amount;
                    }

                    // Yesterday's cost
                    if end == opt.as_of && opt.as_of != opt.first_day_of_this_month {
                        c.latest_daily_cost_increase = amount;
                    }

                    // Weekly cost increase
                    if last_week {
                        c.latest_weekly_cost_increase += amount;
                    }
                }
            }
        }

        next_token = output.next_page_token().map(|s| s.to_string());
        if next_token.is_none() {
            break;
        }
    }

    Ok(costs)
}

struct CeInput {
    time_period: DateInterval,
    group_by: Vec<GroupDefinition>,
    filter: Expression,
}

fn build_cost_explorer_input(opt: &GetCostsOption, account_ids: &[String]) -> CeInput {
    let time_period = DateInterval::builder()
        .start(&opt.first_day_of_last_month)
        .end(&opt.as_of)
        .build()
        .unwrap();

    let group_by = vec![GroupDefinition::builder()
        .r#type(GroupDefinitionType::Dimension)
        .key(CE_COST_GROUP_BY)
        .build()];

    // Build filter: account IDs AND exclude record types
    let account_filter = Expression::builder()
        .dimensions(
            DimensionValues::builder()
                .key(aws_sdk_costexplorer::types::Dimension::LinkedAccount)
                .set_values(Some(account_ids.to_vec()))
                .build(),
        )
        .build();

    let mut and_expressions = vec![account_filter];

    // Exclude record types
    let mut exclude_values = Vec::new();
    if opt.exclude_credit {
        exclude_values.push("Credit".to_string());
    }
    if opt.exclude_upfront {
        exclude_values.push("Upfront".to_string());
    }
    if opt.exclude_refund {
        exclude_values.push("Refund".to_string());
    }
    if opt.exclude_support {
        exclude_values.push("Support".to_string());
    }

    if !exclude_values.is_empty() {
        let exclude_expr = Expression::builder()
            .not(
                Expression::builder()
                    .dimensions(
                        DimensionValues::builder()
                            .key(aws_sdk_costexplorer::types::Dimension::RecordType)
                            .set_values(Some(exclude_values))
                            .build(),
                    )
                    .build(),
            )
            .build();
        and_expressions.push(exclude_expr);
    }

    let filter = Expression::builder()
        .set_and(Some(and_expressions))
        .build();

    CeInput {
        time_period,
        group_by,
        filter,
    }
}
