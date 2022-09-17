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
	costExplorerClient *costexplorer.Client
)

func init() {
	costExplorerClient = costexplorer.NewFromConfig(cfg)
}

// Cost represents a cost for a given account.
type Cost struct {
	AccountID       string
	AccountName     string
	Amount          float64
	AmountYesterday float64
	AmountLastMonth float64
	AmountThisMonth float64
}

type Costs map[string]Cost // map[accountId]Cost

// GetCostsOptions represents options for GetCosts.
type GetCostsOptions struct {
	ExcludeCredit  bool
	ExcludeUpfront bool
	ExcludeRefund  bool
	ExcludeSupport bool
}

// GetCosts returns the costs for given accounts.
func GetCosts(ctx context.Context, accounts Accounts, opt *GetCostsOptions) (Costs, error) {
	accountIds := make([]string, len(accounts))
	i := 0
	for key := range accounts {
		accountIds[i] = key
		i++
	}

	// acos shows by default - THIS_MONTH, vs YESTERDAY, and LAST_MONTH
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
		Granularity: "DAILY",
		Metrics:     []string{"UnblendedCost"},
		TimePeriod:  timePeriod,
		GroupBy: []types.GroupDefinition{
			{
				Type: types.GroupDefinitionTypeDimension,
				Key:  aws.String("LINKED_ACCOUNT"),
			},
		},
		Filter: &types.Expression{
			And: []types.Expression{
				{
					Dimensions: &types.DimensionValues{
						Key:    "LINKED_ACCOUNT",
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

	out, err := costExplorerClient.GetCostAndUsage(ctx, in)
	if err != nil {
		return nil, err
	}
	cs := make(map[string]*Cost)

	// The "range out.ResultsByTime" loop below assumes that the "out.ResultsByTime" slice items are alredy sorted by "r.TimePeriod.Start"
	isThisMonth := false
	for _, r := range out.ResultsByTime {
		for _, g := range r.Groups {
			accntId := g.Keys[0]
			amount, _ := strconv.ParseFloat(*g.Metrics["UnblendedCost"].Amount, 32)
			c, ok := cs[accntId]
			if !ok {
				c = &Cost{
					AccountID:   accntId,
					AccountName: accounts[accntId].Name,
				}
				cs[accntId] = c
			}

			// There's no monthly bill increase from yesterday on the first day of month.
			if *r.TimePeriod.End == today && today != firstDayOfThisMonth {
				c.AmountYesterday = amount
			}

			if *r.TimePeriod.Start == firstDayOfThisMonth {
				isThisMonth = true
			}

			if isThisMonth {
				c.AmountThisMonth += amount
			} else {
				c.AmountLastMonth += amount
			}
		}
	}

	costs := make(map[string]Cost)
	for _, c := range cs {
		costs[c.AccountID] = *c
	}
	return costs, nil
}
