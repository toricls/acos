package acos

import (
	"reflect"
	"testing"
)

func toPointer(s string) *string {
	return &s
}

func TestAccounts_AccountIds(t *testing.T) {
	tests := []struct {
		name string
		a    *Accounts
		want []string
	}{
		{
			name: "empty",
			a:    &Accounts{},
			want: []string{},
		},
		{
			name: "one",
			a: &Accounts{
				"123456789012": Account{
					Id:   toPointer("123456789012"),
					Name: toPointer("test"),
				},
			},
			want: []string{"123456789012"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.AccountIds(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Accounts.GetAccountIds() = %v, want %v", got, tt.want)
			}
		})
	}
}
