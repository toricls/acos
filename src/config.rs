use aws_config::SdkConfig;

/// Load the default AWS SDK configuration.
pub async fn load_aws_config() -> SdkConfig {
    aws_config::load_defaults(aws_config::BehaviorVersion::latest()).await
}
