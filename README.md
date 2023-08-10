# acos

`acos` is an interactive CLI tool to retrieve and show your AWS costs 💸

![acos demo](acos-demo.gif)

## Prerequisites

`acos` requires the below AWS IAM permissions to retrieve cost related data by default.

- [ce:GetCostAndUsage](https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_GetCostAndUsage.html) *1
- [organizations:ListAccounts](https://docs.aws.amazon.com/organizations/latest/APIReference/API_ListAccounts.html) *2

When `acos` runs with the `--ou` option, it requires the below AWS IAM permission _instead of_ `organizations:ListAccounts`.

- [organizations:ListAccountsForParent](https://docs.aws.amazon.com/organizations/latest/APIReference/API_ListAccountsForParent.html)

*1) In addition to the IAM permission, you may also need to enable Cost Explorer in your AWS account [here](https://console.aws.amazon.com/cost-management/home), and to activate IAM Access to the billing console [here](https://console.aws.amazon.com/billing/home#/account) using the root user credentials. See the docs for [AWS Organizational accounts](https://docs.aws.amazon.com/cost-management/latest/userguide/ce-access.html#ce-iam-users) or for [standalone accounts](https://docs.aws.amazon.com/cost-management/latest/userguide/ce-enable.html) for more details.

*2) `acos` falls back to (1) [sts:GetCallerIdentity](https://docs.aws.amazon.com/STS/latest/APIReference/API_GetCallerIdentity.html) and (2) [iam:ListAccountAliases](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAccountAliases.html) to retrieve your AWS account ID and alias, in case `organizations:ListAccounts` fails. This should happen when the AWS account you're accessing via `acos` is not part of an AWS Organization's organization and/or you don't have enough permissions to use the API.

## Installation

```shell
$ go install github.com/toricls/acos/cmd/acos@latest
```

## Usage

```shell
$ acos --help
Usage of acos:
  -asOf string
    	Optional - The date to retrieve the cost data. The format should be 'YYYY-MM-DD'. The default value is today in UTC.
  -comparedTo string
    	Optional - The cost of this month will be compared to either one of 'YESTERDAY' or 'LAST_WEEK'. (default "YESTERDAY")
  -ou string
    	Optional - The ID of an AWS Organizational Unit (OU) or Root to list direct-children AWS accounts. It starts with 'ou-' or 'r-' prefix.
```

### Accounts within AWS Organization

```shell
$ acos
? Select accounts: 567890123456 - my-prod, 123456789012 - my-sandbox
+--------------+--------------+----------------+------------------+----------------+
|  ACCOUNT ID  | ACCOUNT NAME | THIS MONTH ($) | VS YESTERDAY ($) | LAST MONTH ($) |
+--------------+--------------+----------------+------------------+----------------+
| 567890123456 | my-prod      |    5820.334869 |     + 324.526062 |   10765.384186 |
| 123456789012 | my-sandbox   |       0.038331 |       + 0.002255 |       0.127884 |
+--------------+--------------+----------------+------------------+----------------+
|                       TOTAL |    5820.373201 |     + 324.528317 |   10765.512070 |
+--------------+--------------+----------------+------------------+----------------+
As of 2023-07-18.
```

### Accounts under specific AWS Organizational Unit (OU)

Use `--ou` option to filter selectable AWS accounts by specific OU.

```shell
$ acos --ou ou-xxxx-12345678
? Select accounts: 234567890123 - my-dev, 123456789012 - my-sandbox
+--------------+--------------+----------------+------------------+----------------+
|  ACCOUNT ID  | ACCOUNT NAME | THIS MONTH ($) | VS YESTERDAY ($) | LAST MONTH ($) |
+--------------+--------------+----------------+------------------+----------------+
| 234567890123 | my-dev       |     420.102431 |      + 25.801012 |     980.440598 |
| 123456789012 | my-sandbox   |       0.038331 |       + 0.002255 |       0.127884 |
+--------------+--------------+----------------+------------------+----------------+
|                       TOTAL |     420.140762 |      + 25.803267 |     980.568482 |
+--------------+--------------+----------------+------------------+----------------+
As of 2023-07-18.
```

## Todo

- Add some tests
- ~Support OU-based accounts listing~
- ~Support command arguments~, configuration file, and/or env vars for repeated use
- Support JSON format output for piped commands chaining

## Contribution

1. Fork ([https://github.com/toricls/acos/fork](https://github.com/toricls/acos/fork))
4. Create a feature branch
5. Commit your changes
6. Rebase your local changes against the main branch
7. Create a new Pull Request (use [conventional commits] for the title please)

[conventional commits]: https://www.conventionalcommits.org/en/v1.0.0/

## Licence

Distributed under the [Apache-2.0](./LICENSE) license.

## Author

[Tori](https://github.com/toricls)
