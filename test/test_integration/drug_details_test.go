package test_integration

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

func TestDrugDetails(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()

	if _, err := testData.DataAPI.DrugDetails(1); !api.IsErrNotFound(err) {
		t.Errorf("Expected no results error when fetching non-existant drug details. Got %+v", err)
	}

	query := &api.DrugDetailsQuery{
		NDC:         "",
		GenericName: "Non-existant",
		Route:       "topical",
	}
	if _, err := testData.DataAPI.QueryDrugDetails(query); !api.IsErrNotFound(err) {
		t.Errorf("Expected no results error when fetching non-existant drug details. Got %+v", err)
	}

	details := []*common.DrugDetails{
		{
			NDC:         "",
			Name:        "Another Drug Form 1",
			GenericName: "Another",
			Route:       "Topical",
		},
		{
			NDC:         "",
			Name:        "Another Drug Form 2",
			GenericName: "Another",
			Route:       "Topical",
			Form:        "Two",
		},
		{
			NDC:         "0123456789",
			Name:        "Some Drug",
			GenericName: "Some",
			Route:       "Oral",
		},
	}

	if err := testData.DataAPI.SetDrugDetails(details); err != nil {
		t.Fatal(err)
	}

	drugs, err := testData.DataAPI.ListDrugDetails()
	test.OK(t, err)
	test.Equals(t, len(details), len(drugs))
	for i, d := range drugs {
		test.Equals(t, details[i], d)
	}

	// Make sure exact matches return the expected result
	for _, drug := range details {
		d, err := testData.DataAPI.QueryDrugDetails(&api.DrugDetailsQuery{
			NDC:         drug.NDC,
			GenericName: drug.GenericName,
			Route:       drug.Route,
			Form:        drug.Form,
		})
		test.OK(t, err)
		test.Equals(t, drug, d)
	}

	// A entry with an NDC should only be found if given that exact same NDC
	// (i.e. should not match a generic name query)
	_, err = testData.DataAPI.QueryDrugDetails(&api.DrugDetailsQuery{
		GenericName: "Some",
		Route:       "Oral",
		Form:        "Three",
	})
	test.Equals(t, true, api.IsErrNotFound(err))
}
