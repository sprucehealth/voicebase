package main

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var paymentRequestType = graphql.NewObject(graphql.ObjectConfig{
	Name: "PaymentRequest",
	Interfaces: []*graphql.Interface{
		nodeInterfaceType,
	},
	Fields: graphql.Fields{
		"id":                 &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"requestingEntity":   &graphql.Field{Type: graphql.NewNonNull(entityType)},
		"paymentMethod":      &graphql.Field{Type: paymentMethodInterfaceType},
		"currency":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"amountInCents":      &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"status":             &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"processingError":    &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"requestedTimestamp": &graphql.Field{Type: graphql.Int},
		"completedTimestamp": &graphql.Field{Type: graphql.Int},
		"allowPay":           &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
	},
})

var paymentRequestQuery = &graphql.Field{
	Type: graphql.NewNonNull(paymentRequestType),
	Args: graphql.FieldConfigArgument{
		"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		svc := serviceFromParams(p)
		return lookupPaymentRequest(ctx, svc, ram, p.Args["id"].(string))
	}),
}

func lookupPaymentRequest(ctx context.Context, svc *service, ram raccess.ResourceAccessor, id string) (interface{}, error) {
	resp, err := ram.Payment(ctx, &payments.PaymentRequest{PaymentID: id})
	if grpc.Code(err) == codes.NotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	rPayment, err := transformPaymentToResponse(ctx, resp.Payment, ram, svc.staticURLPrefix)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return rPayment, err
}
