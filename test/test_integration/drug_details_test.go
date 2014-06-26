package test_integration

import (
	"carefront/api"
	"carefront/common"

	"testing"
)

func TestDrugDetails(t *testing.T) {

	testData := SetupIntegrationTest(t)
	defer TearDownIntegrationTest(t, testData)

	_, err := testData.DataApi.DrugDetails("non-existant")
	if err != api.NoRowsError {
		t.Errorf("Expected no results error when fetching non-existant drug details. Got %+v", err)
	}

	drug1 := &common.DrugDetails{
		NDC:  "0123456789",
		Name: "Some Drug",
	}
	drug2 := &common.DrugDetails{
		NDC:  "1122334455",
		Name: "Another Drug",
	}
	details := map[string]*common.DrugDetails{
		drug1.NDC: drug1,
		drug2.NDC: drug2,
	}

	if err := testData.DataApi.SetDrugDetails(details); err != nil {
		t.Errorf("SetDrugDetails failed with %s", err.Error())
	}

	for ndc, drug := range details {
		d, err := testData.DataApi.DrugDetails(ndc)
		if err != nil {
			t.Errorf("DrugDetails failed with %s", err.Error())
		}
		if d.NDC != drug.NDC {
			t.Errorf("Expected ndc %s, got %s", drug.NDC, d.NDC)
		}
		if d.Name != drug.Name {
			t.Errorf("Expected name %s, got %s", drug.Name, d.Name)
		}
	}
}
