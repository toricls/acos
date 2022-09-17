package acos

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
)

var (
	// AWS clients
	organizationsClient *organizations.Client
)

func init() {
	organizationsClient = organizations.NewFromConfig(cfg)
}

type Account struct {
	Id   string
	Name string
}

type Accounts map[string]Account // map[accountId]Account

func ListAccounts(ctx context.Context) (Accounts, error) {
	out, err := organizationsClient.ListAccounts(
		ctx,
		&organizations.ListAccountsInput{},
	)
	if err != nil {
		return nil, err
	}
	accnts := make(map[string]Account)
	for _, acc := range out.Accounts {
		a := Account{
			Id:   *acc.Id,
			Name: *acc.Name,
		}
		accnts[a.Id] = a
	}
	return accnts, nil
}
