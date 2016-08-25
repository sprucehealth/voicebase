package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
)

const (
	paymentProcessorStripe = "STRIPE"
)

// paymentProcessor represents the possible payment processors mapped to payment methods
var paymentProcessor = graphql.NewEnum(
	graphql.EnumConfig{
		Name: "PaymentProcessor",
		Values: graphql.EnumValueConfigMap{
			paymentProcessorStripe: &graphql.EnumValueConfig{
				Value: paymentProcessorStripe,
			},
		},
	},
)

const (
	paymentMethodTypeCard = "CARD"
)

// paymentMethodType represents the possible payment method types
var paymentMethodType = graphql.NewEnum(
	graphql.EnumConfig{
		Name: "PaymentMethodType",
		Values: graphql.EnumValueConfigMap{
			paymentMethodTypeCard: &graphql.EnumValueConfig{
				Value: paymentMethodTypeCard,
			},
		},
	},
)

var paymentMethodInterfaceType = graphql.NewInterface(
	graphql.InterfaceConfig{
		Name: "PaymentMethod",
		Fields: graphql.Fields{
			"id":               &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"type":             &graphql.Field{Type: graphql.NewNonNull(paymentMethodType)},
			"paymentProcessor": &graphql.Field{Type: graphql.NewNonNull(paymentProcessor)},
			"default":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		},
	},
)

func init() {
	// This is done here rather than at declaration time to avoid an unresolvable compile time decleration loop
	paymentMethodInterfaceType.ResolveType = func(value interface{}, info graphql.ResolveInfo) *graphql.Object {
		switch value.(type) {
		case *models.PaymentCard:
			return paymentMethodCardType
		}
		return nil
	}
}

var paymentMethodCardType = graphql.NewObject(graphql.ObjectConfig{
	Name: "PaymentCard",
	Interfaces: []*graphql.Interface{
		// TODO: Node support
		paymentMethodInterfaceType,
	},
	Fields: graphql.Fields{
		"id":                 &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"type":               &graphql.Field{Type: graphql.NewNonNull(paymentMethodType)},
		"default":            &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"paymentProcessor":   &graphql.Field{Type: graphql.NewNonNull(paymentProcessor)},
		"tokenizationMethod": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"brand":              &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"last4":              &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"isApplePay":         &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"isAndroidPay":       &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
	},
})

func resolveEntityPaymentMethods(p graphql.ResolveParams) (interface{}, error) {
	ent := p.Source.(*models.Entity)
	ctx := p.Context
	ram := raccess.ResourceAccess(p)

	resp, err := ram.PaymentMethods(ctx, &payments.PaymentMethodsRequest{
		EntityID: ent.ID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return transformPaymentMethodsToResponse(resp.PaymentMethods), nil
}
