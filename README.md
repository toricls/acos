# acos

`acos` is an interactive CLI tool to retrieve and show your AWS costs using AWS Organizations and AWS Cost Explorer APIs 💸

![acos demo](acos-demo.gif)

## Prerequisites

`acos` by default uses AWS Organizations and AWS Cost Explorer APIs, so that it requires the following AWS IAM permissions to retrieve cost related data.

- [ce:GetCostAndUsage](https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_GetCostAndUsage.html) [^1]
- [organizations:ListAccounts](https://docs.aws.amazon.com/organizations/latest/APIReference/API_ListAccounts.html) [^2]

With the `--ou` option, it requires `organizations:ListAccountsForParent` IAM permission _instead of_ `organizations:ListAccounts`.

- [organizations:ListAccountsForParent](https://docs.aws.amazon.com/organizations/latest/APIReference/API_ListAccountsForParent.html) [^2]

[^1]: Make sure you also have [AWS Cost Explorer](https://console.aws.amazon.com/cost-management/home) enabled and have [IAM access to the billing data](https://console.aws.amazon.com/billing/home#/account) activated using your root user credentials beforehand. See also the [docs to enable Cost Explorer for AWS Organizational accounts](https://docs.aws.amazon.com/cost-management/latest/userguide/ce-access.html#ce-iam-users), and the [docs to activate IAM access to the billing data](https://docs.aws.amazon.com/IAM/latest/UserGuide/tutorial_billing.html).

[^2]: `acos` falls back to using (1) [sts:GetCallerIdentity](https://docs.aws.amazon.com/STS/latest/APIReference/API_GetCallerIdentity.html) and (2) [iam:ListAccountAliases](https://docs.aws.amazon.com/IAM/latest/APIReference/API_ListAccountAliases.html) to retrieve your AWS account ID and alias, in case `organizations:ListAccounts` fails. This should happen when the AWS account you're accessing via `acos` is not part of an AWS Organization, and/or you don't have sufficient permissions to use the AWS Organizations APIs.

## Installation

```shell
$ go install github.com/toricls/acos/cmd/acos@latest
```

## Usage

```shell
$ acos --help
Usage of acos:
  -accountIds string
    	Optional - Comma-separated AWS account IDs to retrieve costs. The interactive account selector is skipped when this flag is set.
  -asOf string
    	Optional - The date to retrieve the cost data. The format should be 'YYYY-MM-DD'. The default value is today in UTC.
  -comparedTo string
    	Optional - The cost of this month will be compared to either one of 'YESTERDAY' or 'LAST_WEEK'. This flag is ignored when the -json flag is set. (default "YESTERDAY")
  -json
    	Optional - Print JSON instead of a table.
  -ou string
    	Optional - The ID of an AWS Organizational Unit (OU) or Root to list direct-children AWS accounts. It must start with 'ou-' or 'r-' prefix. This flag is ignored when the -accountIds flag is set.
```

### Accounts within AWS Organization

```shell
$ acos
? Select accounts: 567890123456 - my-prod, 123456789012 - my-sandbox
+--------------+--------------+----------------+------------------+----------------+
|  ACCOUNT ID  | ACCOUNT NAME | THIS MONTH ($) | VS YESTERDAY ($) | LAST MONTH ($) |
+--------------+--------------+----------------+------------------+----------------+
| 123456789012 | my-sandbox   |       0.038331 |       + 0.002255 |       0.127884 |
| 567890123456 | my-prod      |    5820.334869 |     + 324.526062 |   10765.384186 |
+--------------+--------------+----------------+------------------+----------------+
|                       TOTAL |    5820.373201 |     + 324.528317 |   10765.512070 |
+--------------+--------------+----------------+------------------+----------------+
As of 2023-07-18.
```

### Accounts under specific AWS Organizational Unit (OU)

Use `--ou` option to filter selectable AWS accounts by specific OU. Note that the `--ou` option is ignored when the `--accountIds` option is set.

```shell

```shell
$ acos --ou ou-xxxx-12345678
Retrieving AWS accounts under the OU 'ou-xxxx-12345678'...
? Select accounts: 234567890123 - my-dev, 123456789012 - my-sandbox
+--------------+--------------+----------------+------------------+----------------+
|  ACCOUNT ID  | ACCOUNT NAME | THIS MONTH ($) | VS YESTERDAY ($) | LAST MONTH ($) |
+--------------+--------------+----------------+------------------+----------------+
| 123456789012 | my-sandbox   |       0.038331 |       + 0.002255 |       0.127884 |
| 234567890123 | my-dev       |     420.102431 |      + 25.801012 |     980.440598 |
+--------------+--------------+----------------+------------------+----------------+
|                       TOTAL |     420.140762 |      + 25.803267 |     980.568482 |
+--------------+--------------+----------------+------------------+----------------+
As of 2023-07-18.
```

### Specific AWS accounts

Use `--accountIds` option to retrieve costs for specific AWS accounts.

```shell
% ./dist/acos --accountIds 123456789012,567890123456
Account IDs specified. Retrieving accounts information...
+--------------+--------------+----------------+------------------+----------------+
|  ACCOUNT ID  | ACCOUNT NAME | THIS MONTH ($) | VS YESTERDAY ($) | LAST MONTH ($) |
+--------------+--------------+----------------+------------------+----------------+
| 123456789012 | my-sandbox   |       0.038331 |       + 0.002255 |       0.127884 |
| 567890123456 | my-prod      |    5820.334869 |     + 324.526062 |   10765.384186 |
+--------------+--------------+----------------+------------------+----------------+
|                       TOTAL |    5820.373201 |     + 324.528317 |   10765.512070 |
+--------------+--------------+----------------+------------------+----------------+
As of 2023-07-18.
```

The `--json` option would be the best fit here because the `--accountIds` option skips the interactive account selector. You may use this combination of options to retrieve costs in a machine-readable format on a cron'd regular basis for example.

```shell
# Using --accountIds and --json options
% ./dist/acos --accountIds 123456789012,567890123456 --json | jq .
{
  "AsOf": "2023-07-18T14:18:00.182227402Z",
  "Costs": [
    {
      "AccountID": "123456789012",
      "AccountName": "my-sandbox",
      "LatestDailyCostIncrease": 0.0022556669922,
      "LatestWeeklyCostIncrease": 0.0091031111333,
      "AmountLastMonth": 0.127884116211,
      "AmountThisMonth": 0.0383317958984
    },
    {
      "AccountID": "567890123456",
      "AccountName": "my-prod",
      "LatestDailyCostIncrease": 324.526062214416504,
      "LatestWeeklyCostIncrease": 1621.45464324951172,
      "AmountLastMonth": 10765.3841868930054,
      "AmountThisMonth": 5820.3348696633911
    }
  ]
}
```

## Todo

- Add some tests
- ~Support OU-based accounts listing~ done
- ~Support command arguments~, configuration file, and/or env vars for repeated use
- ~Support JSON format output for piped commands chaining~ done

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
