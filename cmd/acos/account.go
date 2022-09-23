package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/toricls/acos"
)

var accntIdsForDebugging = os.Getenv("COMMA_SEPARATED_ACCOUNT_IDS")

// getAvailableAccounts returns a list of AWS accounts which the caller has access to.
func getAvailableAccounts(ctx context.Context) (acos.Accounts, error) {
	var availableAccnts acos.Accounts
	var err error

	if len(accntIdsForDebugging) > 0 {
		fmt.Fprintln(os.Stderr, "Running using debugging account IDs...")
		_accnts := strings.Split(accntIdsForDebugging, ",")
		availableAccnts = acos.Accounts{}
		for _, id := range _accnts {
			availableAccnts[id] = acos.Account{
				Id:   &id,
				Name: &id,
			}
		}
	} else {
		availableAccnts, err = acos.ListAccounts(ctx)
		if err != nil {
			fallback := false
			if !acos.IsOrganizationEnabled(err) {
				fmt.Fprint(os.Stderr, "This AWS account is not part of AWS Organizations organization. ")
				fallback = true
			} else if !acos.HasPermissionToOrganizationsApi(err) {
				fmt.Fprint(os.Stderr, "You don't have IAM permissions to perform \"organizations:ListAccounts\". ")
				fallback = true
			}
			if fallback {
				fmt.Fprintln(os.Stderr, "Using AWS STS and IAM instead to obtain your AWS account information...")
				availableAccnts, err = getCallerAccount(ctx)
			}
		}
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

// selectAccounts prompts the user to select AWS accounts to retrieve costs.
// It returns an error if the `accnts` arg doesn't contain any Account.
// If the `accnts` arg contains only one Account, it never prompts the user.
func selectAccounts(accnts acos.Accounts) (acos.Accounts, error) {
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
	err := survey.AskOne(q, &selIdx, survey.WithPageSize(10))
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
