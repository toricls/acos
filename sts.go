package acos

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

var (
	// AWS clients
	stsClient *sts.Client
	iamClient *iam.Client
)

func init() {
	stsClient = sts.NewFromConfig(cfg)
	iamClient = iam.NewFromConfig(cfg)
}

// GetCallerAccount returns the account ID and name for the current user session.
func GetCallerAccount(ctx context.Context) ([]string, error) {
	res := make([]string, 2)
	out, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return res, err
	}
	res[0] = *out.Account // Account ID
	// Try to fetch human-readable account name
	out2, err := iamClient.ListAccountAliases(ctx, &iam.ListAccountAliasesInput{})
	if err != nil {
		return res, err
	}
	if len(out2.AccountAliases) > 0 {
		res[1] = out2.AccountAliases[0] // Alias as account name
	} else {
		res[1] = "Name not configured"
	}
	return res, nil
}
