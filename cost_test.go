package acos

import (
	"context"
	"reflect"
	"testing"
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
			name: "got an error when empty accounts",
			args: args{
				ctx:      context.Background(),
				accounts: Accounts{},
				opt:      NewGetCostsOption(),
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
