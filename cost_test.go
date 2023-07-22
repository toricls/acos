package acos

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

func TestGetCosts(t *testing.T) {
	type args struct {
		ctx      context.Context
		accounts Accounts
		opt      AcosGetCostsOption
	}
	tests := []struct {
		name    string
		args    args
		want    Costs
		wantErr bool
	}{
		{
			name: "error when empty accounts",
			args: args{
				ctx:      context.Background(),
				accounts: Accounts{},
				opt:      NewGetCostsOption(time.Now().UTC()),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCosts(tt.args.ctx, tt.args.accounts, tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCosts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCosts() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockGetCostAndUsageAPI func(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)

func (m mockGetCostAndUsageAPI) GetCostAndUsage(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
	return m(ctx, params, optFns...)
}

func TestWithMock_GetCosts(t *testing.T) {
	ceClient = mockGetCostAndUsageAPI(func(ctx context.Context, params *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
		t.Helper()
		out := &costexplorer.GetCostAndUsageOutput{
			ResultsByTime:            []types.ResultByTime{},
			GroupDefinitions:         []types.GroupDefinition{},
			DimensionValueAttributes: []types.DimensionValuesWithAttributes{},
		}
		return out, nil
	})

	type args struct {
		ctx      context.Context
		accounts Accounts
		opt      AcosGetCostsOption
	}
	tests := []struct {
		name    string
		args    args
		want    Costs
		wantErr bool
	}{
		{
			name: "selected account must exists in result", // even GetCostAndUsage API returns no result. This is because newly created account has no billed cost in most cases.
			args: args{
				ctx: context.Background(),
				accounts: Accounts{
					"123456789012": Account{
						Id:   toPointer("123456789012"),
						Name: toPointer("test"),
					},
				},
				opt: NewGetCostsOption(time.Now().UTC()),
			},
			want: Costs{
				"123456789012": Cost{
					AccountID:                "123456789012",
					AccountName:              "test",
					LatestDailyCostIncrease:  0,
					LatestWeeklyCostIncrease: 0,
					AmountLastMonth:          0,
					AmountThisMonth:          0,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCosts(tt.args.ctx, tt.args.accounts, tt.args.opt)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCosts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCosts() = %v, want %v", got, tt.want)
			}
		})
	}
}
