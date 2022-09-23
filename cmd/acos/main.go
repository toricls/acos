package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"

	"github.com/toricls/acos"
)

func main() {
	ctx := context.Background()

	availableAccnts, err := getAvailableAccounts(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	selectedAccnts, err := selectAccounts(availableAccnts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	var costs acos.Costs
	if costs, err = acos.GetCosts(ctx, selectedAccnts, &acos.GetCostsOptions{
		ExcludeCredit:  true,
		ExcludeUpfront: true,
		ExcludeRefund:  false,
		ExcludeSupport: false,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
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
