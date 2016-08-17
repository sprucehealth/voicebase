package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

// connectVendorStripeAccount
type connectVendorStripeAccountInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	EntityID         string `gql:"entityID,nonempty"`
	Code             string `gql:"code,nonempty"`
}

var connectVendorStripeAccountInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ConnectVendorStripeAccountInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"code":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

const (
	connectVendorStripeAccountErrorCodeExpiredCode = "EXPIRED_CODE"
)

var connectVendorStripeAccountErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "ConnectVendorStripeAccountErrorCode",
	Values: graphql.EnumValueConfigMap{
		connectVendorStripeAccountErrorCodeExpiredCode: &graphql.EnumValueConfig{
			Value:       connectVendorStripeAccountErrorCodeExpiredCode,
			Description: "The provided code is expired.",
		},
	},
})

type connectVendorStripeAccountOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var connectVendorStripeAccountOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ConnectVendorStripeAccountPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: createEntityProfileErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*connectVendorStripeAccountOutput)
			return ok
		},
	},
)

var connectVendorStripeAccountMutation = &graphql.Field{
	Type: graphql.NewNonNull(connectVendorStripeAccountOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(connectVendorStripeAccountInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		var in connectVendorStripeAccountInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}
		return connectVendorStripeAccount(p, in)
	}),
}

func connectVendorStripeAccount(p graphql.ResolveParams, in connectVendorStripeAccountInput) (interface{}, error) {
	ram := raccess.ResourceAccess(p)
	ctx := p.Context

	_, err := ram.ConnectVendorAccount(ctx, &payments.ConnectVendorAccountRequest{
		EntityID: in.EntityID,
		Type:     payments.VENDOR_ACCOUNT_TYPE_STRIPE,
		ConnectVendorAccountOneof: &payments.ConnectVendorAccountRequest_StripeRequest{
			StripeRequest: &payments.StripeAccountConnectRequest{Code: in.Code},
		},
	})
	if err != nil {
		return nil, errors.InternalError(ctx, err)
	}

	return &connectVendorStripeAccountOutput{
		ClientMutationID: in.ClientMutationID,
		Success:          true,
	}, nil
}
