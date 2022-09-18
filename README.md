# acos

`acos` is an interactive CLI tool to retrieve and show your AWS costs 💸

## Prerequisites

`acos` requires the below AWS IAM permissions to retrieve cost related data.

- [ce:GetCostAndUsage](https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_GetCostAndUsage.html) [^1]
- [organizations:ListAccounts](https://docs.aws.amazon.com/organizations/latest/APIReference/API_ListAccounts.html) [^2]

[^1]: In addition to the IAM permission, you also need to enable Cost Explorer in your AWS account [here](https://us-east-1.console.aws.amazon.com/billing/home?#/reports). You may also need beforehand to activate IAM Access to the billing console [here](https://console.aws.amazon.com/billing/home#/account) using the root user credentials.
[^2]: `acos` falls back to (1) [sts:GetCallerIdentity](https://docs.aws.amazon.com/STS/latest/APIReference/API_GetCallerIdentity.html) and (2) [iam:ListAccountAliases](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAccountAliases.html) to retrieve your AWS account ID and alias, in case `organizations:ListAccounts` fails. This should happen when the AWS account you're accessing via `acos` is not part of an AWS Organization's organization and/or you don't have enough permissions to use the API.

## Usage

```shell
$ go run ./cmd/acos
? Select accounts: 567890123456 - my-prod, 123456789012 - my-sandbox
+--------------+--------------+----------------+------------------+----------------+
|  ACCOUNT ID  | ACCOUNT NAME | THIS MONTH ($) | VS YESTERDAY ($) | LAST MONTH ($) |
+--------------+--------------+----------------+------------------+----------------+
| 567890123456 | my-prod      |    5820.334869 |     + 324.526062 |   10765.384186 |
| 123456789012 | my-sandbox   |       0.038331 |       + 0.002255 |       0.127884 |
+--------------+--------------+----------------+------------------+----------------+
|                       TOTAL |    5820.373201 |     + 324.528317 |   10765.512070 |
+--------------+--------------+----------------+------------------+----------------+
As of 2022-09-18.
```

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
