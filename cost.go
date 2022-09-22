package acos

import (
	"context"
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

// Cost represents a cost for a given account.
type Cost struct {
	AccountID                string
	AccountName              string
	Amount                   float64
	LatestDailyCostIncreaase float64
	AmountLastMonth          float64
	AmountThisMonth          float64
}

type Costs map[string]Cost // map[accountId]Cost

// GetCostsOptions represents options for GetCosts.
type GetCostsOptions struct {
	ExcludeCredit  bool
	ExcludeUpfront bool
	ExcludeRefund  bool
	ExcludeSupport bool
}

func getAccountId(g *types.Group) string {
	return g.Keys[0]
}

func getAmount(g *types.Group) float64 {
	if f, err := strconv.ParseFloat(*g.Metrics[ceCostMetric].Amount, 32); err == nil {
		return f
	}
	// TODO: debug log the error
	return 0
}

// GetCosts returns the costs for given accounts.
func GetCosts(ctx context.Context, accounts Accounts, opt *GetCostsOptions) (Costs, error) {
	accountIds := make([]string, len(accounts))
	i := 0
	for key := range accounts {
		accountIds[i] = key
		i++
	}

	// acos shows the following by default - THIS_MONTH, vs YESTERDAY, and LAST_MONTH
	dateFmt := "2006-01-02"
	t := time.Now().UTC()
	today := t.Format(dateFmt)
	year, month, _ := t.Date()
	firstDayOfLastMonth := time.Date(year, month-1, 1, 0, 0, 0, 0, t.Location()).Format(dateFmt)
	timePeriod := &types.DateInterval{
		Start: aws.String(firstDayOfLastMonth),
		End:   aws.String(today),
	}
	// Just for calculation at the end of this function.
	firstDayOfThisMonth := time.Date(year, month, 1, 0, 0, 0, 0, t.Location()).Format(dateFmt)

	in := &costexplorer.GetCostAndUsageInput{
		Granularity: ceDataGranularity,
		Metrics:     []string{ceCostMetric},
		TimePeriod:  timePeriod,
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

	if opt == nil {
		// default option
		opt = &GetCostsOptions{
			ExcludeCredit:  true,
			ExcludeUpfront: true,
			ExcludeRefund:  false,
			ExcludeSupport: false,
		}
	}
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

	var nextToken *string
	costs := make(map[string]Cost)
	for {
		in.NextPageToken = nextToken

		out, err := ceClient.GetCostAndUsage(ctx, in)
		if err != nil {
			return nil, err
		}

		thisMonth := false
		for _, r := range out.ResultsByTime {

			if *r.TimePeriod.Start == firstDayOfThisMonth {
				// We assume that the AWS API returns sorted "out.ResultsByTime" slice items
				// by "r.TimePeriod.Start".
				// So we can assume that the rest of the items are also for this month, when
				// the "r.TimePeriod.Start" equals to the "first day of this month",
				thisMonth = true
			}

			for _, g := range r.Groups {
				accntId := getAccountId(&g)
				amount := getAmount(&g)

				c, ok := costs[accntId]
				if !ok {
					c = Cost{
						AccountID:   accntId,
						AccountName: accounts[accntId].Name,
					}
				}

				if *r.TimePeriod.End == today {
					// The types.ResultByTime item which represents yesterday's cost should have
					// today's date in "r.TimePeriod.End", and yesterday's date in "r.TimePeriod.Start".
					// Because we call the AWS API with "DAILY" granularity, we don't need to test the
					// "r.TimePeriod.Start" value.

					if today != firstDayOfThisMonth {
						// But also there's no "current month's bill increase" from yesterday when
						// today is the first day of month.
						c.LatestDailyCostIncreaase = amount
					}
				}

				if thisMonth {
					c.AmountThisMonth += amount
				} else {
					c.AmountLastMonth += amount
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
