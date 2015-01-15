package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

func TestPathways(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()

	pathway := &common.Pathway{
		Tag:            "zombie",
		Name:           "Zombified",
		MedicineBranch: "Voodoo",
		Status:         common.PathwayActive,
	}
	test.OK(t, testData.DataAPI.CreatePathway(pathway))

	p, err := testData.DataAPI.PathwayForTag(pathway.Tag)
	test.OK(t, err)
	test.Equals(t, pathway, p)

	p, err = testData.DataAPI.Pathway(pathway.ID)
	test.OK(t, err)
	test.Equals(t, pathway, p)

	ps, err := testData.DataAPI.ListPathways(false)
	test.OK(t, err)
	test.Equals(t, 2, len(ps)) // Includes the default 'Acne' pathway
}

func TestPathwayMenu(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()

	menu := &common.PathwayMenu{
		Title: "What are you here to see the doctor for today?",
		Items: []*common.PathwayMenuItem{
			{
				Title: "Acne",
				Type:  common.PathwayMenuPathwayType,
				Pathway: &common.Pathway{
					ID:   1,
					Name: "Acne",
				},
			},
			{
				Title: "Anti-aging",
				Type:  common.PathwayMenuSubmenuType,
				SubMenu: &common.PathwayMenu{
					Title: "Getting old? What would you like to see the doctor for?",
					Items: []*common.PathwayMenuItem{
						{
							Title: "Wrinkles",
							Type:  common.PathwayMenuPathwayType,
							Pathway: &common.Pathway{
								ID:   2,
								Name: "Wrinkles",
							},
						},
						{
							Title: "Hair Loss",
							Type:  common.PathwayMenuPathwayType,
							Conditionals: []*common.Conditional{
								{
									Op:    "==",
									Key:   "gender",
									Value: "male",
								},
							},
							Pathway: &common.Pathway{
								ID:   2,
								Name: "Wrinkles",
							},
						},
					},
				},
			},
		},
	}
	test.OK(t, testData.DataAPI.UpdatePathwayMenu(menu))

	menu2, err := testData.DataAPI.PathwayMenu()
	test.OK(t, err)
	test.Equals(t, menu, menu2)
}
