package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
)

var partnerIntegrationType = graphql.NewObject(graphql.ObjectConfig{
	Name:        "PartnerIntegration",
	Description: "Represents the state of a partner integration",
	Fields: graphql.Fields{
		"buttonText": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"buttonURL":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"title":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"subtitle":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"connected":  &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errored":    &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
	},
})

func lookupPartnerIntegrationsForOrg(p graphql.ResolveParams, orgID string) ([]*models.PartnerIntegration, error) {
	ctx := p.Context
	ram := raccess.ResourceAccess(p)
	svc := serviceFromParams(p)
	// *** stripe ****

	var errored bool
	stripeIntegration := &models.PartnerIntegration{
		ButtonText: "Connect to Stripe",
		ButtonURL:  svc.stripeConnectURL,
		Title:      "Connect Your Stripe Account",
		Subtitle:   "Allow Spruce to connect your Stripe account to enable patient billpay.",
	}
	resp, err := ram.VendorAccounts(ctx, &payments.VendorAccountsRequest{
		EntityID: orgID,
	})
	if err != nil {
		golog.Errorf("Unable to read vendor account for %s : %s", orgID, err)
		errored = true
	}

	// connected
	if resp != nil && len(resp.VendorAccounts) > 0 {
		stripeIntegration.Connected = true
		stripeIntegration.Title = "Connected to Stripe"
		stripeIntegration.ButtonText = "Stripe Dashboard"
		stripeIntegration.ButtonURL = "https://dashboard.stripe.com"
		stripeIntegration.Subtitle = "View and manage your transaction history through Stripe."
	}

	// errored
	if errored {
		stripeIntegration.Connected = false
		stripeIntegration.Errored = true
		stripeIntegration.Title = "Unable to connect to Stripe"
		stripeIntegration.Subtitle = "Sorry, something went wrong during the connection process, please try connecting again."
	}

	return []*models.PartnerIntegration{stripeIntegration}, nil
}
