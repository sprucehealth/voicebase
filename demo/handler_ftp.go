package demo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
)

type favoriteTreatmentPlanHandler struct {
	dataApi        api.DataAPI
	localServerURL string
}

type favoriteTreatmentPlanRequestData struct {
	FavoriteTreatmentPlanTag string `json:"tag"`
}

func NewFavoriteTreatmentPlanHandler(dataApi api.DataAPI, localServerURL string) *favoriteTreatmentPlanHandler {
	return &favoriteTreatmentPlanHandler{
		dataApi:        dataApi,
		localServerURL: localServerURL,
	}
}

func (f *favoriteTreatmentPlanHandler) IsAuthorized(r *http.Request) (bool, error) {
	return true, nil
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

	authHeader := r.Header.Get("Authorization")

	// ********** STEP 1: first open the case to push it into REVIEWING mode **********
	if err := reviewPatientVisit(patientVisitId, authHeader, r.Host, f.localServerURL); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// ********** STEP 2: pick a treatment plan for the visit **********
	tpResponse, err := pickTreatmentPlan(patientVisitId, authHeader, r.Host, f.localServerURL)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// ********** STEP 3: first add the regimen steps in the context of a patient visit **********

	favoriteTreatmentPlan.RegimenPlan.TreatmentPlanId = tpResponse.TreatmentPlan.Id
	favoriteTreatmentPlan.RegimenPlan, err = addRegimenToTreatmentPlan(favoriteTreatmentPlan.RegimenPlan, authHeader, r.Host, f.localServerURL)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// ********** STEP 4: now add favorite treatment plan to doctor **********

	ftpRequestData := &doctor_treatment_plan.DoctorFavoriteTreatmentPlansRequestData{
		FavoriteTreatmentPlan: favoriteTreatmentPlan,
	}
	jsonData, err := json.Marshal(ftpRequestData)
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
	res, err := http.DefaultClient.Do(addFTPRequest)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		apiservice.WriteError(fmt.Errorf("Expected 200 instead got %d", res.StatusCode), w, r)
		return
	}

	// ********** STEP 5: go ahead and submit this treatment plan to clear this visit out of the doctor's queue **********
	if err := submitTreatmentPlan(tpResponse.TreatmentPlan.Id.Int64(), "foo", authHeader, r.Host, f.localServerURL); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
