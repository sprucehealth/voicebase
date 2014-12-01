package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/apiservice/router"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
)

const defaultBaseURL = "https://http://staging-api.carefront.net"

type DoctorClient struct {
	BaseURL   string
	AuthToken string
}

// Auth signs in as the given doctor account returning the auth response.
// AuthToken is not updated because that could lead to a race condition.
// It is up to the caller to update the struct.
func (dc *DoctorClient) Auth(email, password string) (*doctor.AuthenticationResponse, error) {
	var res doctor.AuthenticationResponse
	err := dc.do("POST", router.DoctorAuthenticateURLPath, nil,
		doctor.AuthenticationRequestData{
			Email:    email,
			Password: password,
		}, &res, nil)
	return &res, err
}

// UpdateTreatmentPlanNote sets the personalized note for a treatment plan.
func (dc *DoctorClient) UpdateTreatmentPlanNote(treatmentPlanID int64, note string) error {
	return dc.do("PUT", router.DoctorSavedNoteURLPath, nil,
		doctor_treatment_plan.DoctorSavedNoteRequestData{
			TreatmentPlanID: treatmentPlanID,
			Message:         note,
		}, nil, nil)
}

// TreatmentPlan fetches the doctor's view of a treatment plan given an ID.
func (dc *DoctorClient) TreatmentPlan(id int64, abridged bool) (*common.TreatmentPlan, error) {
	var res doctor_treatment_plan.DoctorTreatmentPlanResponse
	params := url.Values{"treatment_plan_id": []string{strconv.FormatInt(id, 10)}}
	if abridged {
		params.Set("abridged", "true")
	}
	err := dc.do("GET", router.DoctorTreatmentPlansURLPath, params, nil, &res, nil)
	if err != nil {
		return nil, err
	}
	return res.TreatmentPlan, nil
}

func (dc *DoctorClient) DeleteTreatmentPlan(id int64) error {
	return dc.do("DELETE", router.DoctorTreatmentPlansURLPath,
		url.Values{"treatment_plan_id": []string{strconv.FormatInt(id, 10)}},
		nil, nil, nil)
}

func (dc *DoctorClient) PickTreatmentPlanForVisit(visitID int64, ftp *common.FavoriteTreatmentPlan) (*common.TreatmentPlan, error) {
	req := &doctor_treatment_plan.TreatmentPlanRequestData{
		TPParent: &common.TreatmentPlanParent{
			ParentId:   encoding.NewObjectId(visitID),
			ParentType: common.TPParentTypePatientVisit,
		},
	}
	if ftp != nil {
		req.TPContentSource = &common.TreatmentPlanContentSource{
			Type: common.TPContentSourceTypeFTP,
			ID:   ftp.Id,
		}
	}
	var res doctor_treatment_plan.DoctorTreatmentPlanResponse
	if err := dc.do("POST", router.DoctorTreatmentPlansURLPath, nil, req, &res, nil); err != nil {
		return nil, err
	}
	return res.TreatmentPlan, nil
}

func (dc *DoctorClient) ListFavoriteTreatmentPlans() ([]*common.FavoriteTreatmentPlan, error) {
	var res doctor_treatment_plan.DoctorFavoriteTreatmentPlansResponseData
	err := dc.do("GET", router.DoctorFTPURLPath, nil, nil, &res, nil)
	if err != nil {
		return nil, err
	}
	return res.FavoriteTreatmentPlans, nil
}

func (dc *DoctorClient) CreateFavoriteTreatmentPlan(ftp *common.FavoriteTreatmentPlan) (*common.FavoriteTreatmentPlan, error) {
	return dc.CreateFavoriteTreatmentPlanFromTreatmentPlan(ftp, 0)
}

func (dc *DoctorClient) CreateFavoriteTreatmentPlanFromTreatmentPlan(ftp *common.FavoriteTreatmentPlan, tpID int64) (*common.FavoriteTreatmentPlan, error) {
	var res doctor_treatment_plan.DoctorFavoriteTreatmentPlansResponseData
	err := dc.do("POST", router.DoctorFTPURLPath, nil,
		&doctor_treatment_plan.DoctorFavoriteTreatmentPlansRequestData{
			FavoriteTreatmentPlan: ftp,
			TreatmentPlanID:       tpID,
		}, &res, nil)
	if err != nil {
		return nil, err
	}
	return res.FavoriteTreatmentPlan, err
}

func (dc *DoctorClient) UpdateFavoriteTreatmentPlan(ftp *common.FavoriteTreatmentPlan) (*common.FavoriteTreatmentPlan, error) {
	var res doctor_treatment_plan.DoctorFavoriteTreatmentPlansResponseData
	err := dc.do("PUT", router.DoctorFTPURLPath, nil,
		&doctor_treatment_plan.DoctorFavoriteTreatmentPlansRequestData{
			FavoriteTreatmentPlan: ftp,
		}, &res, nil)
	return res.FavoriteTreatmentPlan, err
}

func (dc *DoctorClient) DeleteFavoriteTreatmentPlan(id int64) error {
	return dc.do("DELETE", router.DoctorFTPURLPath,
		url.Values{"favorite_treatment_plan_id": []string{strconv.FormatInt(id, 10)}},
		nil, nil, nil)
}

func (dc *DoctorClient) CreateRegimenPlan(regimen *common.RegimenPlan) (*common.RegimenPlan, error) {
	var res common.RegimenPlan
	if err := dc.do("POST", router.DoctorRegimenURLPath, nil, regimen, &res, nil); err != nil {
		return nil, err
	}
	return &res, nil
}

func (dc *DoctorClient) do(method, path string, params url.Values, req, res interface{}, headers http.Header) error {
	var body io.Reader
	if req != nil {
		if r, ok := req.(io.Reader); ok {
			body = r
		} else if b, ok := req.([]byte); ok {
			body = bytes.NewReader(b)
		} else {
			if headers == nil {
				headers = http.Header{}
			}
			headers.Set("Content-Type", "application/json")
			b := &bytes.Buffer{}
			if err := json.NewEncoder(b).Encode(req); err != nil {
				return err
			}
			body = b
		}
	}

	u := dc.BaseURL + path
	if len(params) != 0 {
		u += "?" + params.Encode()
	}
	httpReq, err := http.NewRequest(method, u, body)
	if err != nil {
		return err
	}
	for k, v := range headers {
		httpReq.Header[k] = v
	}
	if dc.AuthToken != "" {
		httpReq.Header.Set("Authorization", "token "+dc.AuthToken)
	}
	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpRes.Body.Close()

	switch httpRes.StatusCode {
	case http.StatusNotFound:
		return fmt.Errorf("apiclient: API endpoint '%s%s' not found", dc.BaseURL, path)
	case http.StatusMethodNotAllowed:
		return fmt.Errorf("apiclient: method %s not allowed on endpoint '%s'", method, path)
	case http.StatusOK:
		if res != nil {
			return json.NewDecoder(httpRes.Body).Decode(res)
		}
		return nil
	}

	var e apiservice.SpruceError
	if err := json.NewDecoder(httpRes.Body).Decode(&e); err != nil {
		return fmt.Errorf("apiclient: failed to decode error on %d status code: %s", httpRes.StatusCode, err.Error())
	}
	e.HTTPStatusCode = httpRes.StatusCode
	return &e
}
