package acos

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

var (
	// AWS clients
	ceClient *costexplorer.Client
)

func init() {
	ceClient = costexplorer.NewFromConfig(cfg)
}

const (
	ceDataGranularity = "DAILY"
	ceCostMetric      = "UnblendedCost"
	ceCostGroupBy     = "LINKED_ACCOUNT"
)

// GetCostsOptions represents options for GetCosts. The default values are:
// - ExcludeCredit : true
// - ExcludeUpfront: true
// - ExcludeRefund : false
// - ExcludeSupport: false
type GetCostsOptions struct {
	ExcludeCredit  bool
	ExcludeUpfront bool
	ExcludeRefund  bool
	ExcludeSupport bool
}

// Cost represents a cost for a given account.
type Cost struct {
	AccountID               string
	AccountName             string
	LatestDailyCostIncrease float64
	AmountLastMonth         float64
	AmountThisMonth         float64
}

// Costs represents a map of Cost. The map key is the account ID of the respective Cost.
type Costs map[string]Cost // map[accountId]Cost

// Group wraps up AWS Organization Group struct.
type Group types.Group

func (g *Group) getAccountId() string {
	return g.Keys[0]
}

func (g *Group) getAmount() float64 {
	if f, err := strconv.ParseFloat(*g.Metrics[ceCostMetric].Amount, 32); err == nil {
		return f
	}
	// TODO: debug log the error
	return 0
}

// GetCosts returns the costs for given accounts.
// It raises an error when the `accounts` arg doesn't contain any account.
func GetCosts(ctx context.Context, accounts Accounts, opt *GetCostsOptions) (Costs, error) {
	accountIds := accounts.AccountIds()
	if len(accountIds) == 0 {
		return nil, fmt.Errorf("error no account to retrieve cost: GetCosts requires at least one account in Accounts")
	}
	dates := newGetCostsOptionDates(time.Now().UTC())
	input := newGetCostAndUsageInput(opt, dates, accountIds)

	// The following GetCostAndUsage API won't return any result in some cases (e.g. when the account is newly created).
	// We create and fill the result map with the account IDs and names first, and then fill the amount later,
	// to make sure the result map always contains all the accounts.
	costs := make(map[string]Cost)
	for _, a := range accounts {
		costs[*a.Id] = Cost{
			AccountID:               *a.Id,
			AccountName:             *a.Name,
			LatestDailyCostIncrease: 0,
			AmountLastMonth:         0,
			AmountThisMonth:         0,
		}
	}

	var nextToken *string
	for {
		input.NextPageToken = nextToken

		out, err := ceClient.GetCostAndUsage(ctx, &input)
		if err != nil {
			return nil, err
		}

		thisMonth := false
		for _, r := range out.ResultsByTime {

			if *r.TimePeriod.Start == dates.firstDayOfThisMonth {
				// We assume that the AWS API returns sorted "out.ResultsByTime" slice items
				// by "r.TimePeriod.Start".
				// So we can assume that the rest of the items are also for this month, when
				// the "r.TimePeriod.Start" equals to the "first day of this month".
				//
				// See the official doc for more details about the response data structure:
				// https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_GetCostAndUsage.html#API_GetCostAndUsage_ResponseSyntax
				thisMonth = true
			}

			for _, g := range r.Groups {
				grp := Group(g)
				accntId := grp.getAccountId()
				amount := grp.getAmount()

				c := costs[accntId]
				if thisMonth {
					c.AmountThisMonth += amount
				} else {
					c.AmountLastMonth += amount
				}

				if *r.TimePeriod.End == dates.today {
					// The types.ResultByTime item which represents yesterday's cost should have
					// today's date in "r.TimePeriod.End", and yesterday's date in "r.TimePeriod.Start".
					// Because we call the AWS API with "DAILY" granularity, we don't need to test the
					// "r.TimePeriod.Start" value.

					if dates.today != dates.firstDayOfThisMonth {
						// But also there's no "current month's bill increase" from yesterday when
						// today is the first day of month.
						c.LatestDailyCostIncrease = amount
					}
				}

				costs[accntId] = c
			}
		}

		nextToken = out.NextPageToken
		if nextToken == nil {
			break
		}
	}

	return costs, nil
}

// acos requires the following dates to show - THIS_MONTH, vs YESTERDAY, and LAST_MONTH
type getCostsOptionDates struct {
	today               string
	firstDayOfLastMonth string
	firstDayOfThisMonth string // Just for flagging within the sum-up logic
}

func newGetCostsOptionDates(date time.Time) getCostsOptionDates {
	dateFmt := "2006-01-02"
	t := time.Now().UTC()
	year, month, _ := t.Date()
	return getCostsOptionDates{
		today:               t.Format(dateFmt),
		firstDayOfLastMonth: time.Date(year, month-1, 1, 0, 0, 0, 0, t.Location()).Format(dateFmt),
		firstDayOfThisMonth: time.Date(year, month, 1, 0, 0, 0, 0, t.Location()).Format(dateFmt),
	}
}

func newGetCostAndUsageInput(opt *GetCostsOptions, dates getCostsOptionDates, accountIds []string) costexplorer.GetCostAndUsageInput {
	if opt == nil {
		// default option
		opt = &GetCostsOptions{
			ExcludeCredit:  true,
			ExcludeUpfront: true,
			ExcludeRefund:  false,
			ExcludeSupport: false,
		}
	}

	// Base input parameter
	in := costexplorer.GetCostAndUsageInput{
		Granularity: ceDataGranularity,
		Metrics:     []string{ceCostMetric},
		TimePeriod: &types.DateInterval{
			Start: aws.String(dates.firstDayOfLastMonth),
			End:   aws.String(dates.today),
		},
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  aws.String(ceCostGroupBy),
			},
		},
		Filter: &types.Expression{
			And: []types.Expression{
				{
					Dimensions: &types.DimensionValues{
						Key:    ceCostGroupBy,
						Values: accountIds,
					},
				},
			},
		},
	}

	// Exclude options
	v := []string{}
	if opt.ExcludeCredit {
		v = append(v, "Credit")
	}
	if opt.ExcludeUpfront {
		v = append(v, "Upfront")
	}
	if opt.ExcludeRefund {
		v = append(v, "Refund")
	}
	if opt.ExcludeSupport {
		v = append(v, "Support")
	}
	if len(v) > 0 {
		in.Filter.And = append(in.Filter.And, types.Expression{
			Not: &types.Expression{
				Dimensions: &types.DimensionValues{
					Key:    "RECORD_TYPE",
					Values: v,
				},
			},
		})
	}

	return in
}
