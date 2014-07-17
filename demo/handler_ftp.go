package demo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
)

type favoriteTreatmentPlanHandler struct {
	dataApi api.DataAPI
}

type favoriteTreatmentPlanRequestData struct {
	FavoriteTreatmentPlanTag string `json:"tag"`
}

func NewFavoriteTreatmentPlanHandler(dataApi api.DataAPI) *favoriteTreatmentPlanHandler {
	return &favoriteTreatmentPlanHandler{
		dataApi: dataApi,
	}
}

func (f *favoriteTreatmentPlanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		http.NotFound(w, r)
		return
	}

	requestData := &favoriteTreatmentPlanRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if requestData.FavoriteTreatmentPlanTag == "" {
		apiservice.WriteValidationError("Favorite Treatment Plan Tag cannot be empty", w, r)
		return
	}

	favoriteTreatmentPlan := favoriteTreatmentPlans[requestData.FavoriteTreatmentPlanTag]
	if favoriteTreatmentPlan == nil {
		apiservice.WriteValidationError("Favorite Treatment Plan not found based on tag", w, r)
		return
	}

	// ********** STEP 0: get the first pending visit from the doctors queue to use as context to create the FTP **********
	doctorId, err := f.dataApi.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	pendingItems, err := f.dataApi.GetPendingItemsInDoctorQueue(doctorId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patientVisitId := int64(0)
	for _, item := range pendingItems {
		if item.EventType == api.DQEventTypePatientVisit {
			patientVisitId = item.ItemId
			break
		}
	}

	if patientVisitId == 0 {
		apiservice.WriteValidationError("Unable to find a pending patient visit for doctor", w, r)
		return
	}

	// ********** STEP 1: first open the case to push it into REVIEWING mode **********

	visitReviewRequest, err := http.NewRequest("GET", dVisitReviewUrl+"?patient_visit_id="+strconv.FormatInt(patientVisitId, 10), nil)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	visitReviewRequest.Host = r.Host
	visitReviewRequest.Header.Set("Authorization", r.Header.Get("Authorization"))
	res, err := http.DefaultClient.Do(visitReviewRequest)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if res.StatusCode != http.StatusOK {
		apiservice.WriteError(fmt.Errorf("Expected 200 response instead got %d", res.StatusCode), w, r)
		return
	}

	// ********** STEP 2: pick a treatment plan for the visit **********

	jsonData, err := json.Marshal(&doctor_treatment_plan.PickTreatmentPlanRequestData{
		TPParent: &common.TreatmentPlanParent{
			ParentId:   encoding.NewObjectId(patientVisitId),
			ParentType: common.TPParentTypePatientVisit,
		},
	})

	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	pickATPRequest, err := http.NewRequest("POST", dTPUrl, bytes.NewReader(jsonData))
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	pickATPRequest.Host = r.Host
	pickATPRequest.Header.Set("Content-Type", "application/json")
	pickATPRequest.Header.Set("Authorization", r.Header.Get("Authorization"))
	res, err = http.DefaultClient.Do(pickATPRequest)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if res.StatusCode != http.StatusOK {
		apiservice.WriteError(fmt.Errorf("Expected 200 but got %d instead", res.StatusCode), w, r)
		return
	}

	tpResponse := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	if err := json.NewDecoder(res.Body).Decode(tpResponse); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// ********** STEP 3: first add the regimen steps in the context of a patient visit **********

	favoriteTreatmentPlan.RegimenPlan.TreatmentPlanId = tpResponse.TreatmentPlan.Id
	jsonData, err = json.Marshal(favoriteTreatmentPlan.RegimenPlan)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	addRegimenPlanRequest, err := http.NewRequest("POST", regimenUrl, bytes.NewReader(jsonData))
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	addRegimenPlanRequest.Host = r.Host
	addRegimenPlanRequest.Header.Set("Content-Type", "application/json")
	addRegimenPlanRequest.Header.Set("Authorization", r.Header.Get("Authorization"))
	res, err = http.DefaultClient.Do(addRegimenPlanRequest)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if res.StatusCode != http.StatusOK {
		apiservice.WriteError(fmt.Errorf("Expected 200 instead got %d", res.StatusCode), w, r)
		return
	}

	// get updated regimen plan
	regimenPlan := &common.RegimenPlan{}
	if err := json.NewDecoder(res.Body).Decode(&regimenPlan); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	favoriteTreatmentPlan.RegimenPlan = regimenPlan

	// ********** STEP 4: now add favorite treatment plan to doctor **********

	ftpRequestData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansRequestData{
		FavoriteTreatmentPlan: favoriteTreatmentPlan,
	}
	jsonData, err = json.Marshal(ftpRequestData)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	addFTPRequest, err := http.NewRequest("POST", dFavoriteTPUrl, bytes.NewReader(jsonData))
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	addFTPRequest.Host = r.Host
	addFTPRequest.Header.Set("Authorization", r.Header.Get("Authorization"))
	addFTPRequest.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(addFTPRequest)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if res.StatusCode != http.StatusOK {
		apiservice.WriteError(fmt.Errorf("Expected 200 instead got %d", res.StatusCode), w, r)
		return
	}

	// ********** STEP 5: go ahead and submit this treatment plan to clear this visit out of the doctor's queue **********
	jsonData, err = json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanId: tpResponse.TreatmentPlan.Id,
		Message:         "foo",
	})

	submitTPREquest, err := http.NewRequest("PUT", dTPUrl, bytes.NewReader(jsonData))
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	submitTPREquest.Header.Set("Authorization", r.Header.Get("Authorization"))
	submitTPREquest.Header.Set("Content-Type", "application/json")
	submitTPREquest.Host = r.Host
	res, err = http.DefaultClient.Do(submitTPREquest)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if res.StatusCode != http.StatusOK {
		apiservice.WriteError(fmt.Errorf("Expected 200 but got %d", res.StatusCode), w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
