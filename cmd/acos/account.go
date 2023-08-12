package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/toricls/acos"
)

const ERR_AWS_ORGANIZATION_NOT_ENABLED = "This AWS account is not part of AWS Organizations organization. "
const ERR_INFUFFICIENT_IAM_PERMISSIONS = "Failed to perform \"%s\".  Make sure you have enough IAM permissions. See https://github.com/toricls/acos#prerequisites for the details.\n"

type GetAccountsOption struct {
	AccountIds []string
	OuId       string
}

// getAccounts returns a list of AWS accounts which the caller has access to.
func getAccounts(ctx context.Context, opt GetAccountsOption) (acos.Accounts, error) {
	var availableAccnts acos.Accounts
	var err error

	if len(opt.AccountIds) > 0 {
		fmt.Fprintln(os.Stderr, "Account IDs specified. Retrieving accounts information...")
		availableAccnts, err = getAccountsByIds(ctx, opt.AccountIds)
		if !acos.IsOrganizationEnabled(err) {
			fmt.Fprint(os.Stderr, ERR_AWS_ORGANIZATION_NOT_ENABLED)
		} else if !acos.HasPermissionToOrganizationsApi(err) {
			fmt.Fprintf(os.Stderr, ERR_INFUFFICIENT_IAM_PERMISSIONS, "organizations:ListAccounts")
		}
	} else if len(opt.OuId) > 0 {
		fmt.Fprintf(os.Stderr, "Retrieving AWS accounts under the OU '%s'...\n", opt.OuId)
		availableAccnts, err = getAccountsByOu(ctx, opt.OuId)
		if !acos.OuExists(err) {
			// To avoid duplicated error messages, we override the AWS error by our own.
			err = fmt.Errorf("error the OU \"%s\" doesn't exist", opt.OuId)
			// Stop the process here and we don't fall back to using the "getCallerAccount" func
			// because the specified OU ID is not just valid.
			return nil, err
		} else if !acos.IsOrganizationEnabled(err) {
			fmt.Fprint(os.Stderr, ERR_AWS_ORGANIZATION_NOT_ENABLED)
		} else if !acos.HasPermissionToOrganizationsApi(err) {
			fmt.Fprintf(os.Stderr, ERR_INFUFFICIENT_IAM_PERMISSIONS, "organizations:ListAccountsForParent")
		}
	} else {
		availableAccnts, err = getAccountsInOrg(ctx)
		if !acos.IsOrganizationEnabled(err) {
			fmt.Fprint(os.Stderr, ERR_AWS_ORGANIZATION_NOT_ENABLED)
		} else if !acos.HasPermissionToOrganizationsApi(err) {
			fmt.Fprintf(os.Stderr, ERR_INFUFFICIENT_IAM_PERMISSIONS, "organizations:ListAccounts")
		}
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Falling back to using \"sts:GetCallerIdentity\" and \"iam:ListAccountAliases\" to obtain your AWS account information... ")
		availableAccnts, err = getCallerAccount(ctx)
	}
	return availableAccnts, err
}

func getAccountsByIds(ctx context.Context, accountIds []string) (acos.Accounts, error) {
	accounts, err := acos.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	availableAccnts := make(acos.Accounts, len(accountIds))
	for _, id := range accountIds {
		if len(id) == 0 {
			continue
		}
		if a, ok := accounts[id]; ok {
			availableAccnts[id] = acos.Account{
				Id:   a.Id,
				Name: a.Name,
			}
		} else {
			fmt.Fprintf(os.Stderr, "Account ID '%s' is not found in your AWS organization\n", id)
		}
	}
	return availableAccnts, nil
}

func getAccountsByOu(ctx context.Context, ouId string) (acos.Accounts, error) {
	// Should regex the ouId before calling the API?
	// Doc - https://docs.aws.amazon.com/organizations/latest/APIReference/API_ListAccountsForParent.html#organizations-ListAccountsForParent-request-ParentId
	return acos.ListAccountsByOu(ctx, ouId)
}

func getAccountsInOrg(ctx context.Context) (acos.Accounts, error) {
	return acos.ListAccounts(ctx)
}

// getCallerAccount returns the AWS account information of the caller.
//
// This function expects to be used when the caller is not part of AWS Organizations organization,
// or when the caller doesn't have IAM permissions to perform "organizations:ListAccounts".
func getCallerAccount(ctx context.Context) (acos.Accounts, error) {
	accnt, err := acos.GetCallerAccount(ctx)
	if err != nil {
		return nil, err
	}
	acosAccounts := make(acos.Accounts)
	acosAccounts[accnt[0]] = acos.Account{
		Id:   &accnt[0],
		Name: &accnt[1],
	}
	return acosAccounts, nil
}

// promptAccountsSelection prompts the user to select AWS accounts to retrieve costs.
// It returns an error if the `accnts` arg doesn't contain any Account.
// If the `accnts` arg contains only one Account, it never prompts the user.
func promptAccountsSelection(accnts acos.Accounts) (acos.Accounts, error) {
	if len(accnts) == 0 {
		return nil, fmt.Errorf("error no accounts found")
	} else if len(accnts) == 1 {
		// No need to prompt the user to select accounts if there is only one account.
		return accnts, nil
	}

	opts := make([]string, len(accnts))
	accntIds := make([]string, len(accnts))
	i := 0
	for _, a := range accnts {
		opts[i] = fmt.Sprintf("%s - %s", *a.Id, *a.Name)
		accntIds[i] = *a.Id
		i++
	}
	q := &survey.MultiSelect{
		Message:  "Select accounts:",
		Options:  opts,
		PageSize: 10,
	}

	var selIdx []int
	err := survey.AskOne(
		q,
		&selIdx,
		survey.WithPageSize(10),
		survey.WithKeepFilter(true), // It would be useful to keep the typed filter because we assume that people often use some sort of "prefix" for related AWS account names like "myproduct-prod", "myproduct-staging", "my-product-qa", ...
		survey.WithStdio(os.Stdin, os.Stderr, os.Stderr), // Use stderr for the prompt message to avoid messing up the JSON output.
	)
	if err != nil {
		return nil, err
	}
	if len(selIdx) == 0 {
		return nil, fmt.Errorf("error no accounts selected")
	}

	result := make(acos.Accounts)
	for _, v := range selIdx {
		accntId := accntIds[v]
		result[accntId] = accnts[accntId]
	}
	return result, nil
}
