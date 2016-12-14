package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/settings"
)

func TestVisitLayoutsForPatient(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_1",
		Type: auth.AccountType_PATIENT,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	orgID := "orgID"
	categoryID := "categoryID"
	layoutID := "layoutID"
	layoutName := "test"
	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key: baymaxgraphqlsettings.ConfigKeyPatientInitiatedVisits,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
		},
	}, nil))

	g.layoutC.Expect(mock.NewExpectation(g.layoutC.ListVisitCategories, &layout.ListVisitCategoriesRequest{}).WithReturns(
		&layout.ListVisitCategoriesResponse{
			Categories: []*layout.VisitCategory{
				{
					ID: categoryID,
				},
			},
		}, nil))

	g.layoutC.Expect(mock.NewExpectation(g.layoutC.ListVisitLayouts, &layout.ListVisitLayoutsRequest{
		VisitCategoryID: categoryID,
	}).WithReturns(&layout.ListVisitLayoutsResponse{
		VisitLayouts: []*layout.VisitLayout{
			{
				ID:   layoutID,
				Name: layoutName,
			},
		},
	}, nil))

	res := g.query(ctx, `
		query _ {
			visitLayoutsForPatientInitiatedVisits(organizationID: "orgID") {
				items {
					name 
					id
				}
			}
		}`, nil)
	responseEquals(t, `{
	"data": {
		"visitLayoutsForPatientInitiatedVisits": {
			"items": [{
				"name": "test",
				"id": "layoutID"
			}]
		}
	}
}`, res)

}

func TestPatientInitiatedVisitDrafts(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_1",
		Type: auth.AccountType_PATIENT,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	orgID := "orgID"
	entID := "entID"
	visitID := "visitID"

	// Looking up the account's entity for the org
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(
		[]*directory.Entity{
			{
				ID:   entID,
				Type: directory.EntityType_PATIENT,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
				Memberships: []*directory.Entity{
					{ID: orgID, Type: directory.EntityType_ORGANIZATION},
				},
			},
		}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Visits, &care.GetVisitsRequest{
		Submitted:        false,
		Triaged:          false,
		PatientInitiated: true,
		OrganizationID:   orgID,
		Query: &care.GetVisitsRequest_CreatorID{
			CreatorID: entID,
		},
	}).WithReturns(
		&care.GetVisitsResponse{
			Visits: []*care.Visit{
				{
					ID:              visitID,
					Name:            "test",
					Preferences:     &care.Visit_Preference{},
					LayoutVersionID: "layoutVersionID",
				},
			},
		}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgID,
		},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_ORGANIZATION,
			ID:   orgID,
			Info: &directory.EntityInfo{
				DisplayName: "displayName",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.VisitLayoutVersion, &layout.GetVisitLayoutVersionRequest{
		ID: "layoutVersionID",
	}).WithReturns(&layout.GetVisitLayoutVersionResponse{
		VisitLayoutVersion: &layout.VisitLayoutVersion{
			IntakeLayoutLocation: "testlocation",
		},
	}, nil))

	g.layoutStore.Expect(mock.NewExpectation(g.layoutStore.GetIntake, "testlocation").WithReturns(&layout.Intake{}, nil))
	g.ra.Expect(mock.NewExpectation(g.ra.GetAnswersForVisit, &care.GetAnswersForVisitRequest{
		VisitID:              visitID,
		SerializedForPatient: true,
	}).WithReturns(&care.GetAnswersForVisitResponse{}, nil))

	res := g.query(ctx, `
		query _ {
			patientInitiatedVisitDrafts(organizationID: "orgID") {
				items {
					name 
					id
				}
			}
		}`, nil)
	responseEquals(t, `{
	"data": {
		"patientInitiatedVisitDrafts": {
			"items": [{
				"name": "test",
				"id": "visitID"
			}]
		}
	}
}`, res)

}
