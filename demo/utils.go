package demo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
)

func loginAsDoctor(email string, password, apiDomain string) (string, *common.Doctor, error) {
	params := url.Values{
		"email":    []string{email},
		"password": []string{password},
	}
	loginRequest, err := http.NewRequest("POST", LocalServerURL+dAuthUrl, strings.NewReader(params.Encode()))
	if err != nil {
		return "", nil, err
	}
	loginRequest.Host = apiDomain
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(loginRequest)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("Expected 200 response intsead got %d", res.StatusCode)
	}

	responseData := &doctor.DoctorAuthenticationResponse{}
	err = json.NewDecoder(res.Body).Decode(responseData)
	if err != nil {
		return "", nil, err
	}

	return responseData.Token, responseData.Doctor, nil
}

func reviewPatientVisit(patientVisitId int64, authHeader, apiDomain string) error {
	visitReviewRequest, err := http.NewRequest("GET", LocalServerURL+dVisitReviewUrl+"?patient_visit_id="+strconv.FormatInt(patientVisitId, 10), nil)
	if err != nil {
		return err
	}
	visitReviewRequest.Host = apiDomain
	visitReviewRequest.Header.Set("Authorization", authHeader)
	res, err := http.DefaultClient.Do(visitReviewRequest)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200 response instead got %d", res.StatusCode)
	}

	return nil
}

func pickTreatmentPlan(patientVisitId int64, authHeader, apiDomain string) (*doctor_treatment_plan.DoctorTreatmentPlanResponse, error) {
	jsonData, err := json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TPParent: &common.TreatmentPlanParent{
			ParentId:   encoding.NewObjectId(patientVisitId),
			ParentType: common.TPParentTypePatientVisit,
		},
	})
	if err != nil {
		return nil, err
	}

	pickATPRequest, err := http.NewRequest("POST", LocalServerURL+dTPUrl, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	pickATPRequest.Host = apiDomain
	pickATPRequest.Header.Set("Content-Type", "application/json")
	pickATPRequest.Header.Set("Authorization", authHeader)
	res, err := http.DefaultClient.Do(pickATPRequest)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Expected 200 but got %d instead", res.StatusCode)
	}

	tpResponse := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	err = json.NewDecoder(res.Body).Decode(tpResponse)
	if err != nil {
		return nil, err
	}

	return tpResponse, nil
}

func addRegimenToTreatmentPlan(regimenPlan *common.RegimenPlan, authHeader, apiDomain string) (*common.RegimenPlan, error) {
	jsonData, err := json.Marshal(regimenPlan)
	if err != nil {
		return nil, err
	}
	addRegimenPlanRequest, err := http.NewRequest("POST", LocalServerURL+regimenUrl, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	addRegimenPlanRequest.Host = apiDomain
	addRegimenPlanRequest.Header.Set("Content-Type", "application/json")
	addRegimenPlanRequest.Header.Set("Authorization", authHeader)
	res, err := http.DefaultClient.Do(addRegimenPlanRequest)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Expected 200 instead got %d", res.StatusCode)
	}

	updatedRegimenPlan := &common.RegimenPlan{}
	err = json.NewDecoder(res.Body).Decode(&updatedRegimenPlan)

	if err != nil {
		return nil, err
	}

	return updatedRegimenPlan, nil
}

func addTreatmentsToTreatmentPlan(treatments []*common.Treatment, treatmentPlanId int64, authHeader, apiDomain string) error {
	jsonData, err := json.Marshal(doctor_treatment_plan.AddTreatmentsRequestBody{
		Treatments:      treatments,
		TreatmentPlanId: encoding.NewObjectId(treatmentPlanId),
	})
	if err != nil {
		return err
	}

	addTreatmentsRequest, err := http.NewRequest("POST", LocalServerURL+addTreatmentsUrl, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	addTreatmentsRequest.Host = apiDomain
	addTreatmentsRequest.Header.Set("Authorization", authHeader)
	addTreatmentsRequest.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(addTreatmentsRequest)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200 instead got %d", res.StatusCode)
	}
	return nil
}

func submitTreatmentPlan(treatmentPlanId int64, message, authHeader, apiDomain string) error {
	jsonData, err := json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanId: treatmentPlanId,
		Message:         message,
	})

	submitTPREquest, err := http.NewRequest("PUT", LocalServerURL+dTPUrl, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	submitTPREquest.Header.Set("Authorization", authHeader)
	submitTPREquest.Header.Set("Content-Type", "application/json")
	submitTPREquest.Host = apiDomain
	res, err := http.DefaultClient.Do(submitTPREquest)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200 but got %d", res.StatusCode)
	}
	return nil
}
