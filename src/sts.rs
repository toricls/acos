use anyhow::Result;
use aws_sdk_iam::Client as IamClient;
use aws_sdk_sts::Client as StsClient;

/// Get the caller's account ID and name (alias).
/// Returns (account_id, account_name).
pub async fn get_caller_account(
    sts_client: &StsClient,
    iam_client: &IamClient,
) -> Result<(String, String)> {
    let identity = sts_client.get_caller_identity().send().await?;
    let account_id = identity.account().unwrap_or_default().to_string();

    // Try to fetch human-readable account name via IAM alias
    let aliases_output = iam_client.list_account_aliases().send().await?;
    let account_name = if let Some(first) = aliases_output.account_aliases().first() {
        first.clone()
    } else {
        "Name not configured".to_string()
    };

    Ok((account_id, account_name))
}
