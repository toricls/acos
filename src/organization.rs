use std::collections::HashMap;

use anyhow::Result;
use aws_sdk_organizations::Client as OrganizationsClient;

/// Represents an AWS account with its ID and name.
#[derive(Debug, Clone)]
pub struct Account {
    pub id: String,
    pub name: String,
}

/// A map of account ID -> Account.
pub type Accounts = HashMap<String, Account>;

/// Extension trait for Accounts to get account IDs.
pub trait AccountsExt {
    fn account_ids(&self) -> Vec<String>;
}

impl AccountsExt for Accounts {
    fn account_ids(&self) -> Vec<String> {
        self.keys().cloned().collect()
    }
}

/// List all accounts in the AWS Organization.
pub async fn list_accounts(client: &OrganizationsClient) -> Result<Accounts> {
    let mut accounts = HashMap::new();
    let mut next_token: Option<String> = None;

    loop {
        let mut req = client.list_accounts();
        if let Some(token) = &next_token {
            req = req.next_token(token);
        }

        let output = req.send().await?;

        for acc in output.accounts() {
            let id = acc.id().unwrap_or_default().to_string();
            let name = acc.name().unwrap_or_default().to_string();
            accounts.insert(id.clone(), Account { id, name });
        }

        next_token = output.next_token().map(|s| s.to_string());
        if next_token.is_none() {
            break;
        }
    }

    Ok(accounts)
}

/// List accounts under a specific OU (Organizational Unit).
pub async fn list_accounts_by_ou(
    client: &OrganizationsClient,
    ou_id: &str,
) -> Result<Accounts> {
    let mut accounts = HashMap::new();
    let mut next_token: Option<String> = None;

    loop {
        let mut req = client.list_accounts_for_parent().parent_id(ou_id);
        if let Some(token) = &next_token {
            req = req.next_token(token);
        }

        let output = req.send().await?;

        for acc in output.accounts() {
            let id = acc.id().unwrap_or_default().to_string();
            let name = acc.name().unwrap_or_default().to_string();
            accounts.insert(id.clone(), Account { id, name });
        }

        next_token = output.next_token().map(|s| s.to_string());
        if next_token.is_none() {
            break;
        }
    }

    Ok(accounts)
}

/// Check if the error indicates that AWS Organizations is not enabled.
pub fn is_organization_not_enabled(err: &anyhow::Error) -> bool {
    let err_str = format!("{:?}", err);
    err_str.contains("AWSOrganizationsNotInUseException")
}

/// Check if the error indicates insufficient permissions for Organizations API.
pub fn has_no_permission_to_organizations_api(err: &anyhow::Error) -> bool {
    let err_str = format!("{:?}", err);
    err_str.contains("AccessDeniedException")
}

/// Check if the error indicates the OU does not exist.
pub fn ou_not_found(err: &anyhow::Error) -> bool {
    let err_str = format!("{:?}", err);
    err_str.contains("ParentNotFoundException")
}
