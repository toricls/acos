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

// Account wraps up AWS Organization Account struct.
type Account types.Account

type Accounts map[string]Account // map[accountId]Account

// AccountIds returns a list of account IDs.
func (a *Accounts) AccountIds() []string {
	accountIds := make([]string, len(*a))
	i := 0
	for key := range *a {
		accountIds[i] = key
		i++
	}
	return accountIds
}

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
			accnts[*acc.Id] = Account(acc)
		}
		nextToken = out.NextToken
		if nextToken == nil {
			break
		}
	}
	return accnts, nil
}

// ListAccountsByOu returns a list of direct-children AWS accounts of an AWS Organization OU.
func ListAccountsByOu(ctx context.Context, ouId string) (Accounts, error) {
	var nextToken *string
	accnts := make(map[string]Account)
	for {
		out, err := organizationsClient.ListAccountsForParent(
			ctx,
			&organizations.ListAccountsForParentInput{
				ParentId:  &ouId,
				NextToken: nextToken,
			},
		)
		if err != nil {
			return nil, err
		}
		for _, acc := range out.Accounts {
			accnts[*acc.Id] = Account(acc)
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

func HasPermissionToOrganizationsApi(err error) bool {
	var errType *types.AccessDeniedException
	return !errors.As(err, &errType)
}

func OuExists(err error) bool {
	var errType *types.ParentNotFoundException
	return !errors.As(err, &errType)
}
