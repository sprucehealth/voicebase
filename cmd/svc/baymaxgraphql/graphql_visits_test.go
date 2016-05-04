package main

import (
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"

	"testing"
)

func TestVisitCategories(t *testing.T) {
	acc := &auth.Account{ID: "account_12345", Type: auth.AccountType_PROVIDER}
	ctx := context.Background()
	ctx = gqlctx.WithAccount(ctx, acc)
	orgID := "entity_org1"

	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(
		mock.NewExpectation(
			g.ra.Entity,
			orgID,
			[]directory.EntityInformation{directory.EntityInformation_CONTACTS},
			int64(0)).WithReturns(&directory.Entity{
			ID:   orgID,
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "test",
			},
		}, nil))

	g.ra.Expect(
		mock.NewExpectation(
			g.ra.EntityForAccountID,
			orgID,
			acc.ID).WithReturns(&directory.Entity{
			ID:   orgID,
			Type: directory.EntityType_INTERNAL,
			Info: &directory.EntityInfo{
				DisplayName: "test",
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
			},
		}, nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetSAML,
			"2.2.1.SAMLLocation",
		).WithReturns("2.2.1.SAMLLayout", nil))

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
			},
		}, nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetSAML,
			"2.1.1.SAMLLocation",
		).WithReturns("2.1.1.SAMLLayout", nil))

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
			},
		}, nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetSAML,
			"1.1.1.SAMLLocation",
		).WithReturns("1.1.1.SAMLLayout", nil))

	g.layoutStore.Expect(
		mock.NewExpectation(
			g.layoutStore.GetReview,
			"1.1.1.ReviewLayoutLocation",
		).WithReturns(&visitreview.SectionListView{}, nil))

	g.query(ctx, `
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
			     			reviewLayout
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

	// too hard to check output but if the expectations with the mocks are met that is sufficient for now
}
