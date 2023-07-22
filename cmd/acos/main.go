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
	// Flags
	var ouId, asOfStr string
	flag.StringVar(&ouId, "ou", "", "Optional - The ID of an AWS Organizational Unit (OU) or Root to list direct-children AWS accounts. It starts with 'ou-' or 'r-' prefix.")
	flag.StringVar(&asOfStr, "asOf", "", "Optional - The date to retrieve the cost data. The format should be 'YYYY-MM-DD'. The default value is today in UTC.")
	flag.Parse()

	var asOf time.Time
	if len(asOfStr) > 0 {
		if t, err := time.Parse("2006-01-02", asOfStr); err != nil {
			fmt.Fprintln(os.Stderr, "error invalid date format for the --asOf flag. It should be 'YYYY-MM-DD'.")
			os.Exit(3)
		} else {
			asOf = t
		}
	} else {
		asOf = time.Now().UTC()
	}

	// Choose accounts
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

	// Get costs
	var costs acos.Costs
	if costs, err = acos.GetCosts(ctx, selectedAccnts, acos.NewGetCostsOption(asOf)); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(4)
	}

	// Print table
	print(&costs, asOf)
}

func print(costs *acos.Costs, asOf time.Time) {
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
	t.SetCaption(true, fmt.Sprintf("As of %s.", asOf.Format("2006-01-02")))
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
