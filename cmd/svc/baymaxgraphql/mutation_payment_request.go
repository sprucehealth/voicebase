package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// createPaymentRequest
type createPaymentRequestInput struct {
	ClientMutationID   string `gql:"clientMutationId"`
	RequestingEntityID string `gql:"requestingEntityID,nonempty"`
	Amount             int    `gql:"amount,nonempty"`
}

var createPaymentRequestInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CreatePaymentRequestInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId":   newClientMutationIDInputField(),
			"requestingEntityID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"amount":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.Int)},
		},
	},
)

const (
	createPaymentRequestErrorCodeVendorNotFound = "NO_VENDOR_FOR_ENTITY"
)

var createPaymentRequestErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "CreatePaymentRequestErrorCode",
	Values: graphql.EnumValueConfigMap{
		createPaymentRequestErrorCodeVendorNotFound: &graphql.EnumValueConfig{
			Value:       createPaymentRequestErrorCodeVendorNotFound,
			Description: "The requested entity couldn't be matched to a vendor account",
		},
	},
})

type createPaymentRequestOutput struct {
	ClientMutationID string                 `json:"clientMutationId,omitempty"`
	Success          bool                   `json:"success"`
	ErrorCode        string                 `json:"errorCode,omitempty"`
	ErrorMessage     string                 `json:"errorMessage,omitempty"`
	PaymentRequest   *models.PaymentRequest `json:"paymentRequest"`
}

var createPaymentRequestOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CreatePaymentRequestPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: createPaymentRequestErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"paymentRequest":   &graphql.Field{Type: paymentRequestType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*createPaymentRequestOutput)
			return ok
		},
	},
)

var createPaymentRequestMutation = &graphql.Field{
	Type: graphql.NewNonNull(createPaymentRequestOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(createPaymentRequestInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		var in createPaymentRequestInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}
		return createPaymentRequest(p, in)
	}),
}

func createPaymentRequest(p graphql.ResolveParams, in createPaymentRequestInput) (interface{}, error) {
	svc := serviceFromParams(p)
	ctx := p.Context
	ram := raccess.ResourceAccess(p)
	resp, err := ram.CreatePayment(ctx, &payments.CreatePaymentRequest{
		RequestingEntityID: in.RequestingEntityID,
		Amount:             uint64(in.Amount),
		Currency:           "USD", // Always default to this for now
	})
	if grpc.Code(err) == codes.NotFound {
		return &createPaymentRequestOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        createPaymentRequestErrorCodeVendorNotFound,
			ErrorMessage:     fmt.Sprintf("Vendor for %s Not Found", in.RequestingEntityID),
		}, nil
	} else if err != nil {
		return nil, errors.InternalError(p.Context, err)
	}

	paymentRequest, err := transformPaymentToResponse(ctx, resp.Payment, ram, svc.staticURLPrefix)
	if err != nil {
		return nil, errors.InternalError(p.Context, err)
	}
	return &createPaymentRequestOutput{
		ClientMutationID: in.ClientMutationID,
		Success:          true,
		PaymentRequest:   paymentRequest,
	}, nil
}

// acceptPaymentRequest
type acceptPaymentRequestInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	PaymentRequestID string `gql:"paymentRequestID,nonempty"`
	PaymentMethodID  string `gql:"paymentMethodID,nonempty"`
}

var acceptPaymentRequestInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AcceptPaymentRequestInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"paymentRequestID": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"paymentMethodID":  &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		},
	},
)

const (
	acceptPaymentRequestErrorCodeNotFound = "NOT_FOUND"
)

var acceptPaymentRequestErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "AcceptPaymentRequestErrorCode",
	Values: graphql.EnumValueConfigMap{
		acceptPaymentRequestErrorCodeNotFound: &graphql.EnumValueConfig{
			Value:       acceptPaymentRequestErrorCodeNotFound,
			Description: "The requested payment could not be found",
		},
	},
})

type acceptPaymentRequestOutput struct {
	ClientMutationID string                 `json:"clientMutationId,omitempty"`
	Success          bool                   `json:"success"`
	ErrorCode        string                 `json:"errorCode,omitempty"`
	ErrorMessage     string                 `json:"errorMessage,omitempty"`
	PaymentRequest   *models.PaymentRequest `json:"paymentRequest"`
}

var acceptPaymentRequestOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AcceptPaymentRequestPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: acceptPaymentRequestErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"paymentRequest":   &graphql.Field{Type: paymentRequestType},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*acceptPaymentRequestOutput)
			return ok
		},
	},
)

var acceptPaymentRequestMutation = &graphql.Field{
	Type: graphql.NewNonNull(acceptPaymentRequestOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(acceptPaymentRequestInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		var in acceptPaymentRequestInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}
		return acceptPaymentRequest(p, in)
	}),
}

func acceptPaymentRequest(p graphql.ResolveParams, in acceptPaymentRequestInput) (interface{}, error) {
	svc := serviceFromParams(p)
	ctx := p.Context
	ram := raccess.ResourceAccess(p)
	resp, err := ram.AcceptPayment(ctx, &payments.AcceptPaymentRequest{
		PaymentID:       in.PaymentRequestID,
		PaymentMethodID: in.PaymentMethodID,
	})
	if grpc.Code(err) == codes.NotFound {
		return &acceptPaymentRequestOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          false,
			ErrorCode:        acceptPaymentRequestErrorCodeNotFound,
			ErrorMessage:     fmt.Sprintf("Payment %s Not Found", in.PaymentRequestID),
		}, nil
	} else if err != nil {
		return nil, errors.InternalError(p.Context, err)
	}

	paymentRequest, err := transformPaymentToResponse(ctx, resp.Payment, ram, svc.staticURLPrefix)
	if err != nil {
		return nil, errors.InternalError(p.Context, err)
	}
	return &acceptPaymentRequestOutput{
		ClientMutationID: in.ClientMutationID,
		Success:          true,
		PaymentRequest:   paymentRequest,
	}, nil
}
