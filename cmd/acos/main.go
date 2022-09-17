package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cheynewallace/tabby"

	"github.com/toricls/acos"
)

func main() {
	ctx := context.Background()

	var accountIds acos.Accounts
	var err error

	accountIdsForDebugging := os.Getenv("COMMA_SEPARATED_ACCOUNT_IDS")
	if accountIdsForDebugging != "" {
		fmt.Println("Running using debugging account IDs...")
		_accnts := strings.Split(accountIdsForDebugging, ",")
		accountIds = acos.Accounts{}
		for _, id := range _accnts {
			accountIds[id] = acos.Account{
				Id:   id,
				Name: id,
			}
		}
	} else {
		accountIds, err = chooseAccounts(ctx)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	var costs acos.Costs
	if costs, err = acos.GetCosts(ctx, accountIds, &acos.GetCostsOptions{
		ExcludeCredit:  true,
		ExcludeUpfront: true,
		ExcludeRefund:  false,
		ExcludeSupport: false,
	}); err != nil {
		fmt.Println(err.Error())
		return
	}

	t := tabby.New()
	t.AddHeader("AccountID", "AccountName", "This Month", "vs Yesterday", "Last Month")
	for _, c := range costs {
		thisMonth := fmt.Sprintf("%f %s", c.AmountThisMonth, c.Currency)
		positiveOrNegative := "+"
		if c.AmountYesterday < 0.0 {
			positiveOrNegative = "-"
		}
		vsYesterday := fmt.Sprintf("%s%f %s", positiveOrNegative, c.AmountYesterday, c.Currency)
		lastMonth := fmt.Sprintf("%f %s", c.AmountLastMonth, c.Currency)
		t.AddLine(c.AccountID, c.AccountName, thisMonth, vsYesterday, lastMonth)
	}
	t.Print()
}
