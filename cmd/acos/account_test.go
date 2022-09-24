package main

import (
	"reflect"
	"testing"

	"github.com/toricls/acos"
)

func toPointer(s string) *string {
	return &s
}

func Test_selectAccounts(t *testing.T) {
	type args struct {
		accnts acos.Accounts
	}
	tests := []struct {
		name    string
		args    args
		want    acos.Accounts
		wantErr bool
	}{
		{
			name: "error when empty accounts",
			args: args{
				accnts: acos.Accounts{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no prompts when only one account",
			args: args{
				accnts: acos.Accounts{
					"123456789012": acos.Account{
						Id:   toPointer("123456789012"),
						Name: toPointer("test"),
					},
				},
			},
			want: acos.Accounts{
				"123456789012": acos.Account{
					Id:   toPointer("123456789012"),
					Name: toPointer("test"),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectAccounts(tt.args.accnts)
			if (err != nil) != tt.wantErr {
				t.Errorf("selectAccounts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("selectAccounts() = %v, want %v", got, tt.want)
			}
		})
	}
}
