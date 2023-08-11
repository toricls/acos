package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/toricls/acos"
)

const ERR_AWS_ORGANIZATION_NOT_ENABLED = "This AWS account is not part of AWS Organizations organization. "

var accntIdsForDebugging = os.Getenv("COMMA_SEPARATED_ACCOUNT_IDS")

// getAccounts returns a list of AWS accounts which the caller has access to.
func getAccounts(ctx context.Context, ouId string) (acos.Accounts, error) {
	var availableAccnts acos.Accounts
	var err error
	fallback := false

	if len(accntIdsForDebugging) > 0 {
		fmt.Fprintln(os.Stderr, "Running using debugging account IDs...")
		_accnts := strings.Split(accntIdsForDebugging, ",")
		availableAccnts = make(acos.Accounts, len(_accnts))
		for _, id := range _accnts {
			_id := strings.Clone(id)
			availableAccnts[id] = acos.Account{
				Id:   &_id,
				Name: &_id,
			}
		}
	} else if len(ouId) > 0 {
		fmt.Fprintf(os.Stderr, "Retrieving AWS accounts under the OU '%s'...\n", ouId)
		// Should regex the ouId before calling the API?
		// Doc - https://docs.aws.amazon.com/organizations/latest/APIReference/API_ListAccountsForParent.html#organizations-ListAccountsForParent-request-ParentId
		availableAccnts, err = acos.ListAccountsByOu(ctx, ouId)
		if err != nil {
			if !acos.OuExists(err) {
				// To avoid duplicated error messages, we override the AWS error by our own.
				err = fmt.Errorf("error the OU \"%s\" doesn't exist", ouId)
			} else if !acos.IsOrganizationEnabled(err) {
				fmt.Fprint(os.Stderr, ERR_AWS_ORGANIZATION_NOT_ENABLED)
				fallback = true
			} else if !acos.HasPermissionToOrganizationsApi(err) {
				fmt.Fprint(os.Stderr, "You don't have IAM permissions to perform \"organizations:ListAccountsForParent\". ")
				fallback = true
			}
		}
	} else {
		availableAccnts, err = acos.ListAccounts(ctx)
		if err != nil {
			if !acos.IsOrganizationEnabled(err) {
				fmt.Fprint(os.Stderr, ERR_AWS_ORGANIZATION_NOT_ENABLED)
				fallback = true
			} else if !acos.HasPermissionToOrganizationsApi(err) {
				fmt.Fprint(os.Stderr, "You don't have IAM permissions to perform \"organizations:ListAccounts\". ")
				fallback = true
			}
		}
	}

	if fallback {
		fmt.Fprintln(os.Stderr, "Using AWS STS and IAM instead to retrieve your AWS account information... ")
		availableAccnts, err = getCallerAccount(ctx)
	}
	return availableAccnts, err
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
	} else if len(accntIdsForDebugging) > 0 && len(accnts) > 0 {
		// No need to prompt the user to select accounts if this is for debugging.
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
