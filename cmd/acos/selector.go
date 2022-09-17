package main

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/toricls/acos"
)

// chooseAccounts prompts the user to select AWS accounts to retrieve costs.
func chooseAccounts(ctx context.Context) (acos.Accounts, error) {
	accnts, err := acos.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	if len(accnts) == 0 {
		return nil, fmt.Errorf("error no accounts found")
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
		Message: "Choose accounts to show costs:",
		Options: opts,
	}

	var selIdx []int
	err = survey.AskOne(q, &selIdx)
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
