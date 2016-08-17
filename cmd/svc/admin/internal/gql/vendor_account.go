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

const (
	vendorAccountLifecycleUnknown      = "UNKNOWN"
	vendorAccountLifecycleConnected    = "CONNECTED"
	vendorAccountLifecycleDisconnected = "DISCONNECTED"
)

// vendorAccountLifecycle represents the possible lifecycle enum values mapped to vendor accounts
var vendorAccountLifecycle = graphql.NewEnum(
	graphql.EnumConfig{
		Name: "VendorAccountLifecycle",
		Values: graphql.EnumValueConfigMap{
			vendorAccountLifecycleUnknown: &graphql.EnumValueConfig{
				Value: vendorAccountLifecycleUnknown,
			},
			vendorAccountLifecycleConnected: &graphql.EnumValueConfig{
				Value: vendorAccountLifecycleConnected,
			},
			vendorAccountLifecycleDisconnected: &graphql.EnumValueConfig{
				Value: vendorAccountLifecycleDisconnected,
			},
		},
	},
)

const (
	vendorAccountChangeStateUnknown = "UNKNOWN"
	vendorAccountChangeStateNone    = "NONE"
	vendorAccountChangeStatePending = "PENDING"
)

// vendorAccountChangeState represents the possible change state enum values mapped to vendor accounts
var vendorAccountChangeState = graphql.NewEnum(
	graphql.EnumConfig{
		Name: "VendorAccountChangeState",
		Values: graphql.EnumValueConfigMap{
			vendorAccountChangeStateUnknown: &graphql.EnumValueConfig{
				Value: vendorAccountChangeStateUnknown,
			},
			vendorAccountChangeStateNone: &graphql.EnumValueConfig{
				Value: vendorAccountChangeStateNone,
			},
			vendorAccountChangeStatePending: &graphql.EnumValueConfig{
				Value: vendorAccountChangeStatePending,
			},
		},
	},
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
				"lifecycle":   &graphql.Field{Type: graphql.NewNonNull(vendorAccountLifecycle)},
				"changeState": &graphql.Field{Type: graphql.NewNonNull(vendorAccountChangeState)},
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

// updateVendorAccountInput
type updateVendorAccountInput struct {
	VendorAccountID string `gql:"vendorAccountID"`
	Lifecycle       string `gql:"lifecycle"`
	ChangeState     string `gql:"changeState"`
}

var updateVendorAccountInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "UpdateVendorAccountInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"vendorAccountID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"lifecycle":       &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(vendorAccountLifecycle)},
			"changeState":     &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(vendorAccountChangeState)},
		},
	},
)

type updateVendorAccountOutput struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

var updateVendorAccountOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UpdateVendorAccountPayload",
		Fields: graphql.Fields{
			"success":      &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorMessage": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*updateVendorAccountOutput)
			return ok
		},
	},
)

func newUpdateVendorAccountField() *graphql.Field {
	return &graphql.Field{
		Type: graphql.NewNonNull(updateVendorAccountOutputType),
		Args: graphql.FieldConfigArgument{
			common.InputFieldName: &graphql.ArgumentConfig{Type: graphql.NewNonNull(updateVendorAccountInputType)},
		},
		Resolve: updateVendorAccountResolve,
	}
}

func updateVendorAccountResolve(p graphql.ResolveParams) (interface{}, error) {
	var in updateVendorAccountInput
	if err := gqldecode.Decode(p.Args[common.InputFieldName].(map[string]interface{}), &in); err != nil {
		switch err := err.(type) {
		case gqldecode.ErrValidationFailed:
			return nil, errors.Errorf("%s is invalid: %s", err.Field, err.Reason)
		}
		return nil, errors.Trace(err)
	}

	golog.ContextLogger(p.Context).Debugf("Updating Vendor Account - %s: %s, %s", in.VendorAccountID, in.ChangeState, in.Lifecycle)
	lifecycle, err := friendlyVendorAccountLifecycleToService(in.Lifecycle)
	if err != nil {
		return nil, errors.Trace(err)
	}
	changeState, err := friendlyVendorAccountChangeStateToService(in.ChangeState)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if _, err := client.Payments(p).UpdateVendorAccount(p.Context, &payments.UpdateVendorAccountRequest{
		VendorAccountID: in.VendorAccountID,
		Lifecycle:       lifecycle,
		ChangeState:     changeState,
	}); err != nil {
		return nil, errors.Trace(err)
	}

	return &updateVendorAccountOutput{
		Success: true,
	}, nil
}

func friendlyVendorAccountLifecycleToService(al string) (payments.VendorAccountLifecycle, error) {
	switch al {
	case vendorAccountLifecycleUnknown:
		return payments.VENDOR_ACCOUNT_LIFECYCLE_UNKNOWN, nil
	case vendorAccountLifecycleConnected:
		return payments.VENDOR_ACCOUNT_LIFECYCLE_CONNECTED, nil
	case vendorAccountLifecycleDisconnected:
		return payments.VENDOR_ACCOUNT_LIFECYCLE_DISCONNECTED, nil
	}
	return payments.VENDOR_ACCOUNT_LIFECYCLE_UNKNOWN, errors.Errorf("Unknown VendorAccountLifecycle %s", al)
}

func friendlyVendorAccountChangeStateToService(ac string) (payments.VendorAccountChangeState, error) {
	switch ac {
	case vendorAccountChangeStateUnknown:
		return payments.VENDOR_ACCOUNT_CHANGE_STATE_UNKNOWN, nil
	case vendorAccountChangeStateNone:
		return payments.VENDOR_ACCOUNT_CHANGE_STATE_NONE, nil
	case vendorAccountChangeStatePending:
		return payments.VENDOR_ACCOUNT_CHANGE_STATE_PENDING, nil
	}
	return payments.VENDOR_ACCOUNT_CHANGE_STATE_UNKNOWN, errors.Errorf("Unknown VendorAccountChangeState %s", ac)
}
