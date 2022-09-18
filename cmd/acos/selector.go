package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/toricls/acos"
)

// chooseAccounts prompts the user to select AWS accounts to retrieve costs.
func chooseAccounts(ctx context.Context) (acos.Accounts, error) {
	accnts, err := acos.ListAccounts(ctx)

	if err != nil {
		fallback := false
		if !acos.IsOrganizationEnabled(err) {
			fmt.Fprint(os.Stderr, "This AWS account is not part of AWS Organizations organization. ")
			fallback = true
		} else if !acos.HasPermissionToOrganizationsAPI(err) {
			fmt.Fprint(os.Stderr, "You don't have IAM permissions to perform \"organizations:ListAccounts\". ")
			fallback = true
		}
		if fallback {
			fmt.Fprintln(os.Stderr, "Using AWS STS and IAM instead to obtain your AWS account information...")
			accnts, err = getCallerAccount(ctx)
		}
	}

	if err != nil {
		return nil, err
	}
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
		opts[i] = fmt.Sprintf("%s - %s", a.Id, a.Name)
		accntIds[i] = a.Id
		i++
	}
	q := &survey.MultiSelect{
		Message:  "Choose accounts to show costs:",
		Options:  opts,
		PageSize: 10,
	}

	var selIdx []int
	err = survey.AskOne(q, &selIdx, survey.WithPageSize(10))
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

func getCallerAccount(ctx context.Context) (acos.Accounts, error) {
	accnt, err := acos.GetCallerAccount(ctx)
	if err != nil {
		return nil, err
	}
	acosAccounts := make(acos.Accounts)
	acosAccounts[accnt[0]] = acos.Account{
		Id:   accnt[0],
		Name: accnt[1],
	}
	return acosAccounts, nil
}
