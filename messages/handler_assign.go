package messages

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
)

type assignHandler struct {
	dataAPI api.DataAPI
}

func NewAssignHandler(dataAPI api.DataAPI) http.Handler {
	return apiservice.SupportedMethods(&assignHandler{
		dataAPI: dataAPI,
	}, []string{apiservice.HTTP_POST})
}

func (a *assignHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	// only ma or doctor can assign patient case
	if ctxt.Role != api.DOCTOR_ROLE && ctxt.Role != api.MA_ROLE {
		return false, nil
	}

	requestData := &PostMessageRequest{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	doctor, err := a.dataAPI.GetDoctorFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	patientCase, err := a.dataAPI.GetPatientCaseFromId(requestData.CaseID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientCase] = patientCase

	personID, doctorID, err := validateAccess(a.dataAPI, r, patientCase)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PersonId] = personID
	ctxt.RequestCache[apiservice.DoctorId] = doctorID

	return true, nil
}

func (a *assignHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*PostMessageRequest)
	patientCase := ctxt.RequestCache[apiservice.PatientCase].(*common.PatientCase)
	personID := ctxt.RequestCache[apiservice.PersonId].(int64)
	doctorID := ctxt.RequestCache[apiservice.DoctorId].(int64)

	if requestData.CaseID == 0 {
		apiservice.WriteValidationError("case_id is required", w, r)
		return
	}

	// MA can only assign a case that is already claimed
	if ctxt.Role == api.MA_ROLE && patientCase.Status != common.PCStatusClaimed {
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
	case api.MA_ROLE:

		ma = ctxt.RequestCache[apiservice.Doctor].(*common.Doctor)

		// identify the doctor for the case
		assignments, err := a.dataAPI.GetDoctorsAssignedToPatientCase(patientCase.Id.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		for _, doctorAssignment := range assignments {
			if doctorAssignment.Status == api.STATUS_ACTIVE {
				doctor, err = a.dataAPI.GetDoctorFromId(doctorAssignment.ProviderID)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				longDisplayName = doctor.LongDisplayName
				break
			}
		}
	case api.DOCTOR_ROLE:
		doctor = ctxt.RequestCache[apiservice.Doctor].(*common.Doctor)
		ma, err = a.dataAPI.GetMAInClinic()
		if err != nil {
			apiservice.WriteError(err, w, r)
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

	if err := createMessageAndAttachments(msg, requestData.Attachments, personID, doctorID, a.dataAPI, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	dispatch.Default.Publish(&CaseAssignEvent{
		Message: msg,
		Person:  person,
		Case:    patientCase,
		Doctor:  doctor,
		MA:      ma,
	})

	res := &PostMessageResponse{
		MessageID: msg.ID,
	}
	apiservice.WriteJSON(w, res)
}
