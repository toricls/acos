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
	var ouId, asOfStr, comparedTo string
	flag.StringVar(&ouId, "ou", "", "Optional - The ID of an AWS Organizational Unit (OU) or Root to list direct-children AWS accounts. It starts with 'ou-' or 'r-' prefix.")
	flag.StringVar(&asOfStr, "asOf", "", "Optional - The date to retrieve the cost data. The format should be 'YYYY-MM-DD'. The default value is today in UTC.")
	flag.StringVar(&comparedTo, "comparedTo", "YESTERDAY", "Optional - The cost of this month will be compared to either one of 'YESTERDAY' or 'LAST_WEEK'. The default value is 'YESTERDAY'.")
	flag.Parse()

	var asOf time.Time
	if len(asOfStr) > 0 {
		if t, err := time.Parse("2006-01-02", asOfStr); err != nil {
			fmt.Fprintln(os.Stderr, "error invalid date format for the --asOf flag. It should be 'YYYY-MM-DD'.")
			os.Exit(1)
		} else {
			asOf = t
		}
	} else {
		asOf = time.Now().UTC()
	}

	switch comparedTo {
	case "YESTERDAY":
	case "LAST_WEEK":
		break
	default:
		fmt.Fprintln(os.Stderr, "error invalid value for the -comparedTo flag. It should be either 'YESTERDAY' or 'LAST_WEEK'.")
		os.Exit(2)
	}

	// Choose AWS accounts to show costs
	ctx := context.Background()
	accnts, err := getAccounts(ctx, ouId)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(3)
	}
	selectedAccnts, err := promptAccountsSelection(accnts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(4)
	}

	// Get costs
	var costs acos.Costs
	if costs, err = acos.GetCosts(ctx, selectedAccnts, acos.NewGetCostsOption(asOf)); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(5)
	}

	// Print table
	print(&costs, comparedTo, asOf)
}

func print(costs *acos.Costs, comparedTo string, asOf time.Time) {
	// print table
	t := tablewriter.NewWriter(os.Stdout)
	incrHeaderTxt := "vs Yesterday ($)"
	if comparedTo == "LAST_WEEK" {
		incrHeaderTxt = "vs Last Week ($)"
	}
	t.SetHeader([]string{"Account ID", "Account Name", "This Month ($)", incrHeaderTxt, "Last Month ($)"})
	t.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	totalThisMonth, totalIncrease, totalLastMonth := 0.0, 0.0, 0.0
	for _, c := range *costs {
		thisMonth := fmt.Sprintf("%f", c.AmountThisMonth)
		incr := c.LatestDailyCostIncrease
		if comparedTo == "LAST_WEEK" {
			incr = c.LatestWeeklyCostIncrease
		}
		incrStr := fmt.Sprintf("%s %f", getAmountPrefix(incr), incr)
		lastMonth := fmt.Sprintf("%f", c.AmountLastMonth)
		t.Append([]string{c.AccountID, c.AccountName, thisMonth, incrStr, lastMonth})
		totalThisMonth += c.AmountThisMonth
		totalIncrease += incr
		totalLastMonth += c.AmountLastMonth
	}
	t.SetFooter([]string{"", "Total", fmt.Sprintf("%f", totalThisMonth), fmt.Sprintf("%s %f", getAmountPrefix(totalIncrease), totalIncrease), fmt.Sprintf("%f", totalLastMonth)})
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
