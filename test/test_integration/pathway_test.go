package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

func TestPathways(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)

	pathway := &common.Pathway{
		Tag:            "zombie",
		Name:           "Zombified",
		MedicineBranch: "Voodoo",
		Status:         common.PathwayActive,
	}
	test.OK(t, testData.DataAPI.CreatePathway(pathway))

	p, err := testData.DataAPI.PathwayForTag(pathway.Tag, api.PONone)
	test.OK(t, err)
	test.Equals(t, pathway, p)

	p, err = testData.DataAPI.Pathway(pathway.ID, api.PONone)
	test.OK(t, err)
	test.Equals(t, pathway, p)

	ps, err := testData.DataAPI.ListPathways(api.PONone)
	test.OK(t, err)
	test.Equals(t, 2, len(ps)) // Includes the default 'Acne' pathway

	psm, err := testData.DataAPI.PathwaysForTags([]string{"zombie", "health_condition_acne"}, api.PONone)
	test.OK(t, err)
	test.Equals(t, 2, len(psm))
}

func TestPathwayMenu(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)

	menu := &common.PathwayMenu{
		Title: "What are you here to see the doctor for today?",
		Items: []*common.PathwayMenuItem{
			{
				Title:      "Acne",
				Type:       common.PathwayMenuItemTypePathway,
				PathwayTag: "acne",
			},
			{
				Title: "Anti-aging",
				Type:  common.PathwayMenuItemTypeMenu,
				Menu: &common.PathwayMenu{
					Title: "Getting old? What would you like to see the doctor for?",
					Items: []*common.PathwayMenuItem{
						{
							Title:      "Wrinkles",
							Type:       common.PathwayMenuItemTypePathway,
							PathwayTag: "wrinkles",
						},
						{
							Title: "Hair Loss",
							Type:  common.PathwayMenuItemTypePathway,
							Conditionals: []*common.Conditional{
								{
									Op:    "==",
									Key:   "gender",
									Value: "male",
								},
							},
							PathwayTag: "hair_loss",
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

func TestPathwaySTP(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close(t)

	// create pathway
	pathway := &common.Pathway{
		Name:           "test",
		Tag:            "test",
		MedicineBranch: "test",
		Status:         common.PathwayActive,
	}

	test.OK(t, testData.DataAPI.CreatePathway(pathway))

	expectedString := `
	{
		"message" : "hi"
	}`
	test.OK(t, testData.DataAPI.CreatePathwaySTP(pathway.Tag, []byte(expectedString)))

	pathwaySTPData, err := testData.DataAPI.PathwaySTP(pathway.Tag)
	test.OK(t, err)
	test.Equals(t, expectedString, string(pathwaySTPData))

}
