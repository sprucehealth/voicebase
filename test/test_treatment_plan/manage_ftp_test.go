package test_treatment_plan

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestManageFTP(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	admin := test_integration.CreateRandomAdmin(t, testData)
	dr, _, _ := test_integration.SignupRandomTestDoctor(t, testData)

	treatment1 := &common.Treatment{
		DrugDBIds: map[string]string{
			erx.LexiDrugSynId:     "1234",
			erx.LexiGenProductId:  "12345",
			erx.LexiSynonymTypeId: "123556",
			erx.NDC:               "2415",
		},
		DrugInternalName:        "Teting (This - Drug)",
		DosageStrength:          "10 mg",
		DispenseValue:           5,
		DispenseUnitDescription: "Tablet",
		DispenseUnitId:          encoding.NewObjectId(19),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
		PatientInstructions: "Take once daily",
		OTC:                 false,
	}

	regimenSections := []*common.RegimenSection{
		&common.RegimenSection{
			Name: "Morning",
			Steps: []*common.DoctorInstructionItem{
				&common.DoctorInstructionItem{
					Text: "Step 1",
				},
				&common.DoctorInstructionItem{
					Text: "Step 2",
				},
			},
		},
		&common.RegimenSection{
			Name: "Nighttime",
			Steps: []*common.DoctorInstructionItem{
				&common.DoctorInstructionItem{
					Text: "Step 1",
				},
				&common.DoctorInstructionItem{
					Text: "Step 2",
				},
			},
		},
	}

	favoriteTreatmentPlan := &common.FavoriteTreatmentPlan{
		Name: "FTP TEST 1",
		RegimenPlan: &common.RegimenPlan{
			Sections: regimenSections,
		},
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{treatment1},
		},
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"doctor_id":                strconv.FormatInt(dr.DoctorId, 10),
		"favorite_treatment_plans": []*common.FavoriteTreatmentPlan{favoriteTreatmentPlan},
	})
	test.OK(t, err)

	resp, err := testData.AuthPost(testData.APIServer.URL+apipaths.DoctorManageFTPURLPath, "application/json", bytes.NewReader(jsonData), admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	var response struct {
		FavoriteTreatmentPlans []*common.FavoriteTreatmentPlan `json:"favorite_treatment_plans"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	test.OK(t, err)
	test.Equals(t, 1, len(response.FavoriteTreatmentPlans))
	test.Equals(t, true, response.FavoriteTreatmentPlans[0].Id.Int64() > 0)

	/// get ftps for the doctor and ensure they match
	favoriteTreatmentPlans, err := testData.DataApi.GetFavoriteTreatmentPlansForDoctor(dr.DoctorId)
	test.OK(t, err)
	test.Equals(t, 1, len(favoriteTreatmentPlans))
	test.Equals(t, favoriteTreatmentPlans[0].Id.Int64(), response.FavoriteTreatmentPlans[0].Id.Int64())

	// now lets go ahead and modify the FTP to ensure that it registers
	response.FavoriteTreatmentPlans[0].Name = "FTP TEST 3"
	response.FavoriteTreatmentPlans[0].RegimenPlan.Sections = response.FavoriteTreatmentPlans[0].RegimenPlan.Sections[:1]

	// now lets go ahead and make the call to add/modify ftps again
	jsonData, err = json.Marshal(map[string]interface{}{
		"doctor_id":                strconv.FormatInt(dr.DoctorId, 10),
		"favorite_treatment_plans": response.FavoriteTreatmentPlans,
	})
	test.OK(t, err)

	resp, err = testData.AuthPost(testData.APIServer.URL+apipaths.DoctorManageFTPURLPath, "application/json", bytes.NewReader(jsonData), admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	// ensure that the modification was successful
	favoriteTreatmentPlans, err = testData.DataApi.GetFavoriteTreatmentPlansForDoctor(dr.DoctorId)
	test.OK(t, err)
	test.Equals(t, 1, len(favoriteTreatmentPlans))
	test.Equals(t, favoriteTreatmentPlans[0].Id.Int64(), response.FavoriteTreatmentPlans[0].Id.Int64())
	test.Equals(t, response.FavoriteTreatmentPlans[0].Name, favoriteTreatmentPlans[0].Name)
	test.Equals(t, 1, len(favoriteTreatmentPlans[0].RegimenPlan.Sections))

	// now lets go ahead and delete the FTP
	resp, err = testData.AuthDelete(testData.APIServer.URL+apipaths.DoctorManageFTPURLPath+"?doctor_id="+strconv.FormatInt(dr.DoctorId, 10)+"&favorite_treatment_plan_id="+strconv.FormatInt(favoriteTreatmentPlans[0].Id.Int64(), 10), "", nil, admin.AccountId.Int64())
	test.OK(t, err)
	defer resp.Body.Close()
	test.Equals(t, http.StatusOK, resp.StatusCode)

	favoriteTreatmentPlans, err = testData.DataApi.GetFavoriteTreatmentPlansForDoctor(dr.DoctorId)
	test.OK(t, err)
	test.Equals(t, 0, len(favoriteTreatmentPlans))

}
