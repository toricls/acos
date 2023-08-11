package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/olekukonko/tablewriter"

	"github.com/toricls/acos"
)

func main() {
	// Flags
	var ouId, asOfStr, comparedTo string
	var useJson bool
	flag.StringVar(&ouId, "ou", "", "Optional - The ID of an AWS Organizational Unit (OU) or Root to list direct-children AWS accounts. It starts with 'ou-' or 'r-' prefix.")
	flag.StringVar(&asOfStr, "asOf", "", "Optional - The date to retrieve the cost data. The format should be 'YYYY-MM-DD'. The default value is today in UTC.")
	flag.StringVar(&comparedTo, "comparedTo", "YESTERDAY", "Optional - The cost of this month will be compared to either one of 'YESTERDAY' or 'LAST_WEEK'. This flag is ignored when the -json flag is set.")
	flag.BoolVar(&useJson, "json", false, "Optional - Print JSON instead of a table.")
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

	// Sort map keys by AWS Account ID
	keys := make([]string, 0, len(costs))
	for k := range costs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// Map to array
	costArray := make([]acos.Cost, 0, len(costs))
	for _, k := range keys {
		costArray = append(costArray, (costs)[k])
	}

	if useJson {
		if err := printJson(costArray, asOf); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(6)
		}
	} else {
		// Print table
		printTable(costArray, comparedTo, asOf)
	}
}

func printJson(costs []acos.Cost, asOf time.Time) error {
	jsonStr, err := json.Marshal(struct {
		AsOf  time.Time
		Costs []acos.Cost
	}{asOf, costs})
	if err != nil {
		return err
	}
	fmt.Println(string(jsonStr))
	return nil
}

func printTable(costs []acos.Cost, comparedTo string, asOf time.Time) {
	t := tablewriter.NewWriter(os.Stdout)
	incrHeaderTxt := "vs Yesterday ($)"
	if comparedTo == "LAST_WEEK" {
		incrHeaderTxt = "vs Last Week ($)"
	}
	t.SetHeader([]string{"Account ID", "Account Name", "This Month ($)", incrHeaderTxt, "Last Month ($)"})
	t.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	totalThisMonth, totalIncrease, totalLastMonth := 0.0, 0.0, 0.0
	for _, c := range costs {
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
