package main

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

// deletePaymentMethod
type deletePaymentMethodInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	ID               string `gql:"id,nonempty"`
}

var deletePaymentMethodInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "DeletePaymentMethodInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"id":               &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		},
	},
)

const (
	deletePaymentMethodErrorCodeNotFound = "NOT_FOUND"
)

var deletePaymentMethodErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "DeletePaymentMethodErrorCode",
	Values: graphql.EnumValueConfigMap{
		deletePaymentMethodErrorCodeNotFound: &graphql.EnumValueConfig{
			Value:       deletePaymentMethodErrorCodeNotFound,
			Description: "The requested payment method wasn't found",
		},
	},
})

type deletePaymentMethodOutput struct {
	ClientMutationID string                 `json:"clientMutationId,omitempty"`
	Success          bool                   `json:"success"`
	ErrorCode        string                 `json:"errorCode,omitempty"`
	ErrorMessage     string                 `json:"errorMessage,omitempty"`
	PaymentMethods   []models.PaymentMethod `json:"paymentMethods"`
}

var deletePaymentMethodOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "DeletePaymentMethodPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: deletePaymentMethodErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"paymentMethods":   &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(paymentMethodInterfaceType))},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*deletePaymentMethodOutput)
			return ok
		},
	},
)

var deletePaymentMethodMutation = &graphql.Field{
	Type: graphql.NewNonNull(deletePaymentMethodOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(deletePaymentMethodInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		var in deletePaymentMethodInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}
		return deletePaymentMethod(p, in)
	}),
}

func deletePaymentMethod(p graphql.ResolveParams, in deletePaymentMethodInput) (interface{}, error) {
	ctx := p.Context
	ram := raccess.ResourceAccess(p)
	resp, err := ram.DeletePaymentMethod(ctx, &payments.DeletePaymentMethodRequest{
		PaymentMethodID: in.ID,
	})
	if grpc.Code(err) == codes.NotFound {
		return &deletePaymentMethodOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        deletePaymentMethodErrorCodeNotFound,
			ErrorMessage:     fmt.Sprintf("Payment Method %s Not Found", in.ID),
		}, nil
	} else if err != nil {
		return nil, errors.InternalError(p.Context, err)
	}

	return &deletePaymentMethodOutput{
		ClientMutationID: in.ClientMutationID,
		Success:          true,
		PaymentMethods:   transformPaymentMethodsToResponse(resp.PaymentMethods),
	}, nil
}

// addCardPaymentMethod
type addCardPaymentMethodInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	EntityID         string `gql:"entityID,nonempty"`
	PaymentProcessor string `gql:"paymentProcessor,nonempty"`
	Token            string `gql:"token,nonempty"`
}

var addCardPaymentMethodInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AddCardPaymentMethodInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"entityID":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"paymentProcessor": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(paymentProcessor)},
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

// TODO: This currently serves no purpose, but graphql compalins if it's empty
const (
	addCardPaymentMethodErrorCodeMissingInfo = "MISSING_INFO"
)

var addCardPaymentMethodErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "AddCardPaymentMethodErrorCode",
	Values: graphql.EnumValueConfigMap{
		addCardPaymentMethodErrorCodeMissingInfo: &graphql.EnumValueConfig{
			Value:       addCardPaymentMethodErrorCodeMissingInfo,
			Description: "There is missing information",
		},
	},
})

type addCardPaymentMethodOutput struct {
	ClientMutationID string                 `json:"clientMutationId,omitempty"`
	Success          bool                   `json:"success"`
	ErrorCode        string                 `json:"errorCode,omitempty"`
	ErrorMessage     string                 `json:"errorMessage,omitempty"`
	PaymentMethods   []models.PaymentMethod `json:"paymentMethods"`
}

var addCardPaymentMethodOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AddCardPaymentMethodPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: addCardPaymentMethodErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"paymentMethods":   &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(paymentMethodInterfaceType))},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*addCardPaymentMethodOutput)
			return ok
		},
	},
)

var addCardPaymentMethodMutation = &graphql.Field{
	Type: graphql.NewNonNull(addCardPaymentMethodOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(addCardPaymentMethodInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		var in addCardPaymentMethodInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}
		return addCardPaymentMethod(p, in)
	}),
}

func addCardPaymentMethod(p graphql.ResolveParams, in addCardPaymentMethodInput) (interface{}, error) {
	ctx := p.Context
	ram := raccess.ResourceAccess(p)
	req := &payments.CreatePaymentMethodRequest{
		EntityID: in.EntityID,
		Default:  true, // TODO: This currently has no effect
		Type:     payments.PAYMENT_METHOD_TYPE_CARD,
	}
	switch in.PaymentProcessor {
	case paymentProcessorStripe:
		req.StorageType = payments.PAYMENT_METHOD_STORAGE_TYPE_STRIPE
		req.CreatePaymentMethodOneof = &payments.CreatePaymentMethodRequest_StripeCard{
			StripeCard: &payments.StripeCardCreateRequest{
				Token: in.Token,
			},
		}
	default:
		return nil, gqlerrors.FormatError(fmt.Errorf("paymentProcessor %s is unsupported", in.PaymentProcessor))
	}
	resp, err := ram.CreatePaymentMethod(ctx, req)
	if err != nil {
		return nil, errors.InternalError(p.Context, err)
	}
	return &addCardPaymentMethodOutput{
		ClientMutationID: in.ClientMutationID,
		Success:          true,
		PaymentMethods:   transformPaymentMethodsToResponse(resp.PaymentMethods),
	}, nil
}
