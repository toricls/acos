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

type CeGetCostAndUsageAPI interface {
	GetCostAndUsage(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
}

var (
	// AWS clients
	ceClient CeGetCostAndUsageAPI
)

func init() {
	ceClient = costexplorer.NewFromConfig(cfg)
}

const (
	ceDataGranularity = "DAILY"
	ceCostMetric      = "UnblendedCost"
	ceCostGroupBy     = "LINKED_ACCOUNT"
)

// AcosGetCostsOption represents options for GetCosts. The default values are:
// - ExcludeCredit : true
// - ExcludeUpfront: true
// - ExcludeRefund : false
// - ExcludeSupport: false
type AcosGetCostsOption struct {
	ExcludeCredit  bool
	ExcludeUpfront bool
	ExcludeRefund  bool
	ExcludeSupport bool

	// acos requires the following dates to show - THIS_MONTH, vs YESTERDAY, vs LAST_WEEK, and LAST_MONTH
	dates struct {
		asOf                string
		oneWeekAgo          string
		firstDayOfLastMonth string
		firstDayOfThisMonth string // Just for flagging within the sum-up logic
	}
}

func NewGetCostsOption(asOfInUTC time.Time) AcosGetCostsOption {
	opt := AcosGetCostsOption{
		ExcludeCredit:  true,
		ExcludeUpfront: true,
		ExcludeRefund:  false,
		ExcludeSupport: false,
	}

	oneWeekAgo := asOfInUTC.Add(time.Duration(-7) * 24 * time.Hour)
	year, month, _ := asOfInUTC.Date()
	firstDayOfThisMonth := time.Date(year, month, 1, 0, 0, 0, 0, asOfInUTC.Location())
	firstDayOfLastMonth := time.Date(year, month-1, 1, 0, 0, 0, 0, asOfInUTC.Location())

	dateFmt := "2006-01-02" // Use the same format as the AWS API response, "types.ResultByTime.TimePeriod.Start/End".
	opt.dates.asOf = asOfInUTC.Format(dateFmt)
	opt.dates.oneWeekAgo = oneWeekAgo.Format(dateFmt)
	opt.dates.firstDayOfThisMonth = firstDayOfThisMonth.Format(dateFmt)
	opt.dates.firstDayOfLastMonth = firstDayOfLastMonth.Format(dateFmt)
	return opt
}

// Cost represents a cost for a given account.
type Cost struct {
	AccountID                string
	AccountName              string
	LatestDailyCostIncrease  float64
	LatestWeeklyCostIncrease float64
	AmountLastMonth          float64
	AmountThisMonth          float64
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
func GetCosts(ctx context.Context, accounts Accounts, opt AcosGetCostsOption) (Costs, error) {
	accountIds := accounts.AccountIds()
	if len(accountIds) == 0 {
		return nil, fmt.Errorf("error no account to retrieve cost: GetCosts requires at least one account in Accounts")
	}
	ceOpt := acosOptToCostExplorerOpt(opt, accountIds)

	// The following GetCostAndUsage API won't return any result in some cases (e.g. when the account is newly created).
	// We create and fill the result map with the account IDs and names first, and then fill the amount later,
	// to make sure the result map always contains all the accounts.
	costs := make(map[string]Cost)
	for _, a := range accounts {
		costs[*a.Id] = Cost{
			AccountID:                *a.Id,
			AccountName:              *a.Name,
			LatestDailyCostIncrease:  0,
			LatestWeeklyCostIncrease: 0,
			AmountLastMonth:          0,
			AmountThisMonth:          0,
		}
	}

	var nextToken *string
	for {
		ceOpt.NextPageToken = nextToken

		out, err := ceClient.GetCostAndUsage(ctx, &ceOpt)
		if err != nil {
			return nil, err
		}

		thisMonth := false
		lastWeek := false
		for _, r := range out.ResultsByTime {

			if *r.TimePeriod.Start == opt.dates.firstDayOfThisMonth {
				// We assume that the AWS API returns sorted "out.ResultsByTime" slice items
				// by "r.TimePeriod.Start".
				// So we can assume that the rest of the items are also for this month, when
				// the "r.TimePeriod.Start" equals to the "first day of this month".
				//
				// See the official doc for more details about the response data structure:
				// https://docs.aws.amazon.com/aws-cost-management/latest/APIReference/API_GetCostAndUsage.html#API_GetCostAndUsage_ResponseSyntax
				thisMonth = true
			}
			// Flag "lastWeek" if the first week has passed of this month.
			if thisMonth && *r.TimePeriod.Start == opt.dates.oneWeekAgo {
				lastWeek = true
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

				// Store yesterday's cost as the "latest daily cost increase".
				//
				// The types.ResultByTime item, that represents yesterday's cost, should has
				// today's date in "r.TimePeriod.End", and yesterday's date in "r.TimePeriod.Start".
				// We only check the "r.TimePeriod.End" value here, because we we called the AWS API
				// with the "DAILY" granularity.
				if *r.TimePeriod.End == opt.dates.asOf {
					if opt.dates.asOf != opt.dates.firstDayOfThisMonth { // Unless today is the first day of month.
						c.LatestDailyCostIncrease = amount
					}
				}

				// Add the cost onto the "latest weekly cost increase".
				if lastWeek {
					c.LatestWeeklyCostIncrease += amount
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

// acosOptToCostExplorerOpt returns the AWS Cost Explorer's GetCostAndUsageInput param built from the acos options.
func acosOptToCostExplorerOpt(opt AcosGetCostsOption, accountIds []string) costexplorer.GetCostAndUsageInput {
	// Base input parameter
	in := costexplorer.GetCostAndUsageInput{
		Granularity: ceDataGranularity,
		Metrics:     []string{ceCostMetric},
		TimePeriod: &types.DateInterval{
			// Get the cost for the last month and this month
			Start: aws.String(opt.dates.firstDayOfLastMonth),
			End:   aws.String(opt.dates.asOf),
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
