package gql

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/admin/internal/common"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/client"
	"github.com/sprucehealth/backend/cmd/svc/admin/internal/gql/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
)

// newVendorAccountType returns a type object representing a payments vendor account
func newVendorAccountType() *graphql.Object {
	return graphql.NewObject(
		graphql.ObjectConfig{
			Name: "VendorAccount",
			Fields: graphql.Fields{
				"id":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"type":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"accountID":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"lifecycle":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"changeState": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"live":        &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			},
		})
}

func getEntityVendorAccounts(ctx context.Context, paymentsClient payments.PaymentsClient, entityID string) ([]*models.VendorAccount, error) {
	resp, err := paymentsClient.VendorAccounts(ctx, &payments.VendorAccountsRequest{
		EntityID: entityID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return models.TransformVendorAccountsToModel(resp.VendorAccounts), nil
}

// disconnectVendorAccountInput
type disconnectVendorAccountInput struct {
	VendorAccountID string `gql:"vendorAccountID"`
}

var disconnectVendorAccountInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "DisconnectVendorAccountInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"vendorAccountID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

type disconnectVendorAccountOutput struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

var disconnectVendorAccountOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "DisconnectVendorAccountPayload",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*disconnectVendorAccountOutput)
			return ok
		},
	},
)

func newDisconnectVendorAccountField() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(disconnectVendorAccountOutputType),
		Args: graphql.FieldConfigArgument{
			common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(disconnectVendorAccountInputType)},
		},
		Resolve: disconnectVendorAccountResolve,
	}
}

func disconnectVendorAccountResolve(p graphql.ResolveParams) (interface{}, error) {
	var in disconnectVendorAccountInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	golog.ContextLogger(p.Context).Debugf("Disconnecting Vendor Account - %s", in.VendorAccountID)
	if _, err := client.Payments(p).DisconnectVendorAccount(p.Context, &payments.DisconnectVendorAccountRequest{
		VendorAccountID: in.VendorAccountID,
	}); err != nil {
		return nil, errors.Trace(err)
	}

	return &disconnectVendorAccountOutput{
		Success: true,
	}, nil
}
