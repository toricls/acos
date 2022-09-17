package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"

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

	// print table
	t := tablewriter.NewWriter(os.Stdout)
	t.SetHeader([]string{"Account ID", "Account Name", "This Month ($)", "vs Yesterday ($)", "Last Month ($)"})
	t.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	for _, c := range costs {
		thisMonth := fmt.Sprintf("%f", c.AmountThisMonth)
		vsYesterday := fmt.Sprintf("%s %f", getAmountPrefix(c.AmountYesterday), c.AmountYesterday)
		lastMonth := fmt.Sprintf("%f", c.AmountLastMonth)
		t.Append([]string{c.AccountID, c.AccountName, thisMonth, vsYesterday, lastMonth})
	}
	t.Render()
}

func getAmountPrefix(amount float64) string {
	if amount > 0.0 {
		return "+"
	} else if amount < 0.0 {
		return "-"
	}
	return ""
}
