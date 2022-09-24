package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"

	"github.com/toricls/acos"
)

func main() {
	var ouId string
	flag.StringVar(&ouId, "ou", "", "Optional - The ID of an AWS Organizational Unit (OU) or Root to list direct-children AWS accounts. It starts with 'ou-' or 'r-' prefix.")
	flag.Parse()

	ctx := context.Background()
	accnts, err := getAccounts(ctx, ouId)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	selectedAccnts, err := promptAccountsSelection(accnts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	var costs acos.Costs
	if costs, err = acos.GetCosts(ctx, selectedAccnts, acos.NewGetCostsOption()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}

	// print table
	print(&costs)
}

func print(costs *acos.Costs) {
	// print table
	t := tablewriter.NewWriter(os.Stdout)
	t.SetHeader([]string{"Account ID", "Account Name", "This Month ($)", "vs Yesterday ($)", "Last Month ($)"})
	t.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	totalThisMonth, totalYesterday, totalLastMonth := 0.0, 0.0, 0.0
	for _, c := range *costs {
		thisMonth := fmt.Sprintf("%f", c.AmountThisMonth)
		vsYesterday := fmt.Sprintf("%s %f", getAmountPrefix(c.LatestDailyCostIncrease), c.LatestDailyCostIncrease)
		lastMonth := fmt.Sprintf("%f", c.AmountLastMonth)
		t.Append([]string{c.AccountID, c.AccountName, thisMonth, vsYesterday, lastMonth})
		totalThisMonth += c.AmountThisMonth
		totalYesterday += c.LatestDailyCostIncrease
		totalLastMonth += c.AmountLastMonth
	}
	t.SetFooter([]string{"", "Total", fmt.Sprintf("%f", totalThisMonth), fmt.Sprintf("%s %f", getAmountPrefix(totalYesterday), totalYesterday), fmt.Sprintf("%f", totalLastMonth)})
	t.SetFooterAlignment(tablewriter.ALIGN_RIGHT)
	t.SetCaption(true, fmt.Sprintf("As of %s.", time.Now().Format("2006-01-02")))
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
