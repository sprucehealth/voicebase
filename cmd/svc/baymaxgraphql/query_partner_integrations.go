package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
	var stripeIntegration *models.PartnerIntegration

	stripeIntegration = &models.PartnerIntegration{
		ButtonText: "Connect to Stripe",
		ButtonURL:  svc.stripeConnectURL,
		Title:      "Connect Your Stripe Account",
		Subtitle:   "Allow Spruce to connect your Stripe account to enable patient billpay.",
	}
	resp, err := ram.VendorAccounts(ctx, &payments.VendorAccountsRequest{
		EntityID: orgID,
	})
	if err != nil {
		// errored
		golog.Errorf("Unable to read vendor account for %s : %s", orgID, err)
		stripeIntegration.Connected = false
		stripeIntegration.Errored = true
		stripeIntegration.Title = "Unable to connect to Stripe"
		stripeIntegration.Subtitle = "Sorry, something went wrong during the connection process, please try connecting again."
	}

	// connected
	if resp != nil && len(resp.VendorAccounts) > 0 {
		stripeIntegration.Connected = true
		stripeIntegration.Title = "Connected to Stripe"
		stripeIntegration.ButtonText = "Stripe Dashboard"
		stripeIntegration.ButtonURL = "https://dashboard.stripe.com"
		stripeIntegration.Subtitle = "View and manage your transaction history through Stripe."
	}

	// *** hint ***

	hintIntegration := &models.PartnerIntegration{
		ButtonText: "Connect to Hint",
		ButtonURL:  svc.hintConnectURL,
		Title:      "Connect your Hint Account",
		Subtitle:   "Import all patients from Hint into Spruce. Before doing this, contact Spruce Support to configure standard or secure conversations for all patients.",
	}
	patientSyncResp, err := ram.LookupPatientSyncConfiguration(ctx, &patientsync.LookupSyncConfigurationRequest{
		OrganizationEntityID: orgID,
		Source:               patientsync.SOURCE_HINT,
	})
	if err != nil && grpc.Code(err) != codes.NotFound {
		// errored
		golog.Errorf("Unable to lookup sync configuration for org %s: %s", orgID, err)
		hintIntegration.Connected = false
		hintIntegration.Errored = true
		hintIntegration.Title = "Unable to connect to Hint"
		hintIntegration.Subtitle = "Sorry, something went wrong during the connection process, please try connecting again."
	}

	if patientSyncResp != nil {
		hintIntegration.Connected = true
		hintIntegration.Title = "Connected to Hint"
		hintIntegration.ButtonText = "Hint Dashboard"
		hintIntegration.ButtonURL = *flagHintConnectURL
		hintIntegration.Subtitle = "View and manage patient membership information in Hint."
	}

	partnerIntegrations := make([]*models.PartnerIntegration, 0, 2)
	if stripeIntegration != nil {
		partnerIntegrations = append(partnerIntegrations, stripeIntegration)
	}
	partnerIntegrations = append(partnerIntegrations, hintIntegration)

	return partnerIntegrations, nil
}
