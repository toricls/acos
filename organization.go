package acos

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
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

// ListAccounts returns a list of AWS accounts within an AWS Organization organization.
func ListAccounts(ctx context.Context) (Accounts, error) {
	var nextToken *string
	accnts := make(map[string]Account)
	for {
		out, err := organizationsClient.ListAccounts(
			ctx,
			&organizations.ListAccountsInput{
				NextToken: nextToken,
			},
		)
		if err != nil {
			return nil, err
		}
		for _, acc := range out.Accounts {
			a := Account{
				Id:   *acc.Id,
				Name: *acc.Name,
			}
			accnts[a.Id] = a
		}
		nextToken = out.NextToken
		if nextToken == nil {
			break
		}
	}
	return accnts, nil
}

func IsOrganizationEnabled(err error) bool {
	var errType *types.AWSOrganizationsNotInUseException
	return !errors.As(err, &errType)
}

func HasPermissionToOrganizationsAPI(err error) bool {
	var errType *types.AccessDeniedException
	return !errors.As(err, &errType)
}
