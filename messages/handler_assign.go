package messages

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
)

type assignHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

func NewAssignHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&assignHandler{
					dataAPI:    dataAPI,
					dispatcher: dispatcher,
				}),
			api.RoleDoctor, api.RoleCC),
		httputil.Post)
}

func (a *assignHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	requestData := &PostMessageRequest{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctor, err := a.dataAPI.GetDoctorFromAccountID(ctxt.AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	patientCase, err := a.dataAPI.GetPatientCaseFromID(requestData.CaseID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientCase] = patientCase

	personID, doctorID, err := validateAccess(a.dataAPI, r, patientCase)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PersonID] = personID
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	return true, nil
}

func (a *assignHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*PostMessageRequest)
	patientCase := ctxt.RequestCache[apiservice.PatientCase].(*common.PatientCase)
	personID := ctxt.RequestCache[apiservice.PersonID].(int64)
	doctorID := ctxt.RequestCache[apiservice.DoctorID].(int64)

	if requestData.CaseID == 0 {
		apiservice.WriteValidationError("case_id is required", w, r)
		return
	}

	// MA can only assign a case that is already claimed
	if ctxt.Role == api.RoleCC && !patientCase.Claimed {
		apiservice.WriteValidationError("Care coordinator cannot assign a case to a doctor for a case that is not currently claimed by a doctor", w, r)
		return
	}

	people, err := a.dataAPI.GetPeople([]int64{personID})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	person := people[personID]

	var longDisplayName string
	var doctor *common.Doctor
	var ma *common.Doctor
	switch ctxt.Role {
	case api.RoleCC:
		ma = ctxt.RequestCache[apiservice.Doctor].(*common.Doctor)

		// identify the doctor for the case
		assignments, err := a.dataAPI.GetDoctorsAssignedToPatientCase(patientCase.ID.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		for _, doctorAssignment := range assignments {
			if doctorAssignment.Status == api.StatusActive {
				doctor, err = a.dataAPI.GetDoctorFromID(doctorAssignment.ProviderID)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				longDisplayName = doctor.LongDisplayName
				break
			}
		}
	case api.RoleDoctor:
		doctor = ctxt.RequestCache[apiservice.Doctor].(*common.Doctor)

		careTeam, err := a.dataAPI.GetActiveMembersOfCareTeamForCase(patientCase.ID.Int64(), false)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		for _, cp := range careTeam {
			if cp.ProviderRole == api.RoleCC {
				ma, err = a.dataAPI.Doctor(cp.ProviderID, true)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				break
			}
		}

		if ma == nil {
			apiservice.WriteError(fmt.Errorf("No CC assigned to case %d", patientCase.ID.Int64()), w, r)
			return
		}

		longDisplayName = ma.LongDisplayName
	}

	msg := &common.CaseMessage{
		CaseID:    requestData.CaseID,
		PersonID:  personID,
		Body:      requestData.Message,
		IsPrivate: true,
		EventText: fmt.Sprintf("assigned to %s", longDisplayName),
	}

	if err := CreateMessageAndAttachments(msg, requestData.Attachments, personID, doctorID, ctxt.Role, a.dataAPI); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	a.dispatcher.Publish(&CaseAssignEvent{
		Message: msg,
		Person:  person,
		Case:    patientCase,
		Doctor:  doctor,
		MA:      ma,
	})

	res := &PostMessageResponse{
		MessageID: msg.ID,
	}
	httputil.JSONResponse(w, http.StatusOK, res)
}
