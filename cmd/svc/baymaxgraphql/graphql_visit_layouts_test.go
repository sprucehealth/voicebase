package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
)

func TestVisitCategories(t *testing.T) {
	acc := &auth.Account{ID: "account_12345", Type: auth.AccountType_PROVIDER}
	ctx := context.Background()
	ctx = gqlctx.WithAccount(ctx, acc)
	orgID := "entity_org1"

	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns([]*directory.Entity{
		{
			ID:   orgID,
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "test",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		{
			ID:   "entity",
			Type: directory.EntityType_INTERNAL,
			Info: &directory.EntityInfo{
				DisplayName: "test",
			},
			Memberships: []*directory.Entity{
				{
					ID:   orgID,
					Type: directory.EntityType_ORGANIZATION,
				},
			},
		},
	}, nil))

	g.settingsC.Expect(
		mock.NewExpectation(
			g.settingsC.GetValues,
			&settings.GetValuesRequest{
				NodeID: orgID,
				Keys: []*settings.ConfigKey{
					{
						Key: baymaxgraphqlsettings.ConfigKeyVisitAttachments,
					},
				},
			},
		).WithReturns(&settings.GetValuesResponse{
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

	g.layoutC.Expect(
		mock.NewExpectation(
			g.layoutC.ListVisitCategories,
			&layout.ListVisitCategoriesRequest{},
		).WithReturns(&layout.ListVisitCategoriesResponse{
			Categories: []*layout.VisitCategory{
				{
					ID:   "1",
					Name: "B",
				},
				{
					ID:   "2",
					Name: "A",
				},
			},
		}, nil))

	g.layoutC.Expect(
		mock.NewExpectation(
			g.layoutC.ListVisitLayouts,
			&layout.ListVisitLayoutsRequest{
				VisitCategoryID: "2",
			}).WithReturns(&layout.ListVisitLayoutsResponse{
			VisitLayouts: []*layout.VisitLayout{
				{
					ID:   "2.1",
					Name: "D",
				},
				{
					ID:   "2.2",
					Name: "c",
				},
			},
		}, nil))

	g.layoutC.Expect(
		mock.NewExpectation(
			g.layoutC.GetVisitLayoutVersion,
			&layout.GetVisitLayoutVersionRequest{
				VisitLayoutID: "2.2",
			},
		).WithReturns(&layout.GetVisitLayoutVersionResponse{
			VisitLayoutVersion: &layout.VisitLayoutVersion{
				ID:                   "2.2.1",
				SAMLLocation:         "2.2.1.SAMLLocation",
				ReviewLayoutLocation: "2.2.1.ReviewLayoutLocation",
				IntakeLayoutLocation: "2.2.1.IntakeLayoutLocation",
			},
		}, nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetSAML,
			"2.2.1.SAMLLocation",
		).WithReturns("2.2.1.SAMLLayout", nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetIntake,
			"2.2.1.IntakeLayoutLocation",
		).WithReturns(&layout.Intake{}, nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetReview,
			"2.2.1.ReviewLayoutLocation",
		).WithReturns(&visitreview.SectionListView{}, nil))

	g.layoutC.Expect(
		mock.NewExpectation(
			g.layoutC.GetVisitLayoutVersion,
			&layout.GetVisitLayoutVersionRequest{
				VisitLayoutID: "2.1",
			},
		).WithReturns(&layout.GetVisitLayoutVersionResponse{
			VisitLayoutVersion: &layout.VisitLayoutVersion{
				ID:                   "2.1.1",
				SAMLLocation:         "2.1.1.SAMLLocation",
				ReviewLayoutLocation: "2.1.1.ReviewLayoutLocation",
				IntakeLayoutLocation: "2.1.1.IntakeLayoutLocation",
			},
		}, nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetSAML,
			"2.1.1.SAMLLocation",
		).WithReturns("2.1.1.SAMLLayout", nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetIntake,
			"2.1.1.IntakeLayoutLocation",
		).WithReturns(&layout.Intake{}, nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetReview,
			"2.1.1.ReviewLayoutLocation",
		).WithReturns(&visitreview.SectionListView{}, nil))

	g.layoutC.Expect(
		mock.NewExpectation(
			g.layoutC.ListVisitLayouts,
			&layout.ListVisitLayoutsRequest{
				VisitCategoryID: "1",
			}).WithReturns(&layout.ListVisitLayoutsResponse{
			VisitLayouts: []*layout.VisitLayout{
				{
					ID:   "1.1",
					Name: "e",
				},
			},
		}, nil))

	g.layoutC.Expect(
		mock.NewExpectation(
			g.layoutC.GetVisitLayoutVersion,
			&layout.GetVisitLayoutVersionRequest{
				VisitLayoutID: "1.1",
			},
		).WithReturns(&layout.GetVisitLayoutVersionResponse{
			VisitLayoutVersion: &layout.VisitLayoutVersion{
				ID:                   "1.1.1",
				SAMLLocation:         "1.1.1.SAMLLocation",
				ReviewLayoutLocation: "1.1.1.ReviewLayoutLocation",
				IntakeLayoutLocation: "1.1.1.IntakeLayoutLocation",
			},
		}, nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetSAML,
			"1.1.1.SAMLLocation",
		).WithReturns("1.1.1.SAMLLayout", nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetIntake,
			"1.1.1.IntakeLayoutLocation",
		).WithReturns(&layout.Intake{}, nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetReview,
			"1.1.1.ReviewLayoutLocation",
		).WithReturns(&visitreview.SectionListView{}, nil))

	res := g.query(ctx, `
 query _ {
   organization(id: "entity_org1") {
	visitCategories(first: 100) {
	     	edges {
		     	node {
			     	id
			     	name
			     	visitLayouts(first: 100) {
			     		edges {
			     			node {
			     		id
			     		name
			     		version {
			     			samlLayout
			     			layoutPreview
			     		}
			     		}
			     		}
			     	}
				}
			}
		 }
	    }
 }
`, nil)

	responseEquals(t, `{"data":{"organization":{"visitCategories":{"edges":[{"node":{"id":"2","name":"A","visitLayouts":{"edges":[{"node":{"id":"2.2","name":"c","version":{"layoutPreview":"{\"sections\":[],\"type\":\"d_visit_review:sections_list\"}","samlLayout":"\"2.2.1.SAMLLayout\""}}},{"node":{"id":"2.1","name":"D","version":{"layoutPreview":"{\"sections\":[],\"type\":\"d_visit_review:sections_list\"}","samlLayout":"\"2.1.1.SAMLLayout\""}}}]}}},{"node":{"id":"1","name":"B","visitLayouts":{"edges":[{"node":{"id":"1.1","name":"e","version":{"layoutPreview":"{\"sections\":[],\"type\":\"d_visit_review:sections_list\"}","samlLayout":"\"1.1.1.SAMLLayout\""}}}]}}}]}}}}`, res)
}
