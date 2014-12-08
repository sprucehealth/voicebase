package apiclient

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/sprucehealth/backend/doctor_queue"

	"github.com/sprucehealth/backend/apiservice/apipaths"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/messages"
)

const defaultBaseURL = "https://staging-api.carefront.net"

type DoctorClient struct {
	BaseURL    string
	AuthToken  string
	HostHeader string
}

// Auth signs in as the given doctor account returning the auth response.
// AuthToken is not updated because that could lead to a race condition.
// It is up to the caller to update the struct.
func (dc *DoctorClient) Auth(email, password string) (*doctor.AuthenticationResponse, error) {
	var res doctor.AuthenticationResponse
	err := dc.do("POST", apipaths.DoctorAuthenticateURLPath, nil,
		doctor.AuthenticationRequestData{
			Email:    email,
			Password: password,
		}, &res, nil)
	return &res, err
}

// UpdateTreatmentPlanNote sets the personalized note for a treatment plan.
func (dc *DoctorClient) UpdateTreatmentPlanNote(treatmentPlanID int64, note string) error {
	return dc.do("PUT", apipaths.DoctorSavedNoteURLPath, nil,
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
	err := dc.do("GET", apipaths.DoctorTreatmentPlansURLPath, params, nil, &res, nil)
	if err != nil {
		return nil, err
	}
	return res.TreatmentPlan, nil
}

func (dc *DoctorClient) DeleteTreatmentPlan(id int64) error {
	return dc.do("DELETE", apipaths.DoctorTreatmentPlansURLPath,
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
	if err := dc.do("POST", apipaths.DoctorTreatmentPlansURLPath, nil, req, &res, nil); err != nil {
		return nil, err
	}
	return res.TreatmentPlan, nil
}

func (dc *DoctorClient) SubmitTreatmentPlan(treatmentPlanID int64) error {
	return dc.do("PUT", apipaths.DoctorTreatmentPlansURLPath, nil,
		doctor_treatment_plan.TreatmentPlanRequestData{
			TreatmentPlanID: treatmentPlanID,
		}, nil, nil)
}

func (dc *DoctorClient) ListFavoriteTreatmentPlans() ([]*common.FavoriteTreatmentPlan, error) {
	var res doctor_treatment_plan.DoctorFavoriteTreatmentPlansResponseData
	err := dc.do("GET", apipaths.DoctorFTPURLPath, nil, nil, &res, nil)
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
	err := dc.do("POST", apipaths.DoctorFTPURLPath, nil,
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
	err := dc.do("PUT", apipaths.DoctorFTPURLPath, nil,
		&doctor_treatment_plan.DoctorFavoriteTreatmentPlansRequestData{
			FavoriteTreatmentPlan: ftp,
		}, &res, nil)
	return res.FavoriteTreatmentPlan, err
}

func (dc *DoctorClient) DeleteFavoriteTreatmentPlan(id int64) error {
	return dc.do("DELETE", apipaths.DoctorFTPURLPath,
		url.Values{"favorite_treatment_plan_id": []string{strconv.FormatInt(id, 10)}},
		nil, nil, nil)
}

func (dc *DoctorClient) CreateRegimenPlan(regimen *common.RegimenPlan) (*common.RegimenPlan, error) {
	var res common.RegimenPlan
	if err := dc.do("POST", apipaths.DoctorRegimenURLPath, nil, regimen, &res, nil); err != nil {
		return nil, err
	}
	return &res, nil
}

func (dc *DoctorClient) PostCaseMessage(caseID int64, msg string, attachments []*messages.Attachment) (int64, error) {
	var res messages.PostMessageResponse
	err := dc.do("POST", apipaths.CaseMessagesURLPath, nil,
		&messages.PostMessageRequest{
			CaseID:      caseID,
			Message:     msg,
			Attachments: attachments,
		}, &res, nil)
	return res.MessageID, err
}

func (dc *DoctorClient) ListCaseMessages(caseID int64) ([]*messages.Message, []*messages.Participant, error) {
	var res messages.ListResponse
	err := dc.do("GET", apipaths.CaseMessagesListURLPath,
		url.Values{
			"case_id": []string{strconv.FormatInt(caseID, 10)},
		}, nil, &res, nil)
	return res.Items, res.Participants, err
}

func (dc *DoctorClient) AssignCase(caseID int64, msg string, attachments []*messages.Attachment) (int64, error) {
	var res messages.PostMessageResponse
	err := dc.do("POST", apipaths.DoctorAssignCaseURLPath, nil,
		&messages.PostMessageRequest{
			CaseID:      caseID,
			Message:     msg,
			Attachments: attachments,
		}, &res, nil)
	return res.MessageID, err
}

func (dc *DoctorClient) DoctorCaseList() ([]*doctor_queue.PatientsFeedItem, error) {
	var res doctor_queue.PatientsFeedResponse
	err := dc.do("GET", apipaths.DoctorCaseListURLPath, nil, nil, &res, nil)
	return res.Items, err
}

func (dc *DoctorClient) do(method, path string, params url.Values, req, res interface{}, headers http.Header) error {
	return do(dc.BaseURL, dc.AuthToken, dc.HostHeader, method, path, params, req, res, headers)
}
