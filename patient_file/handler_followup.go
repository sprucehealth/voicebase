package patient_file

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/messages"
	patientpkg "github.com/sprucehealth/backend/patient"
)

type followupHandler struct {
	dataAPI            api.DataAPI
	authAPI            api.AuthAPI
	dispatcher         *dispatch.Dispatcher
	expirationDuration time.Duration
	store              storage.Store
}

type followupRequestData struct {
	CaseID int64 `json:"case_id,string"`
}

func NewFollowupHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, expirationDuration time.Duration, dispatcher *dispatch.Dispatcher, store storage.Store) http.Handler {
	return &followupHandler{
		dataAPI:            dataAPI,
		authAPI:            authAPI,
		dispatcher:         dispatcher,
		expirationDuration: expirationDuration,
		store:              store,
	}
}

func (f *followupHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.DOCTOR_ROLE && ctxt.Role != api.MA_ROLE {
		return false, nil
	}

	if r.Method != apiservice.HTTP_POST {
		return false, nil
	}

	var rd followupRequestData
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.RequestData] = rd

	doctorID, err := f.dataAPI.GetDoctorIdFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	patientCase, err := f.dataAPI.GetPatientCaseFromId(rd.CaseID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientCase] = patientCase

	if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorID,
		patientCase.PatientId.Int64(), patientCase.Id.Int64(), f.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (f *followupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patientCase := ctxt.RequestCache[apiservice.PatientCase].(*common.PatientCase)
	doctorID := ctxt.RequestCache[apiservice.DoctorID].(int64)

	patient, err := f.dataAPI.GetPatientFromId(patientCase.PatientId.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctor, err := f.dataAPI.Doctor(doctorID, false)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// first create the followup visit
	followupVisit, err := patientpkg.CreatePendingFollowup(patient, f.dataAPI, f.authAPI, f.dispatcher, f.store, f.expirationDuration)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// now create the message
	msg := &common.CaseMessage{
		CaseID:   patientCase.Id.Int64(),
		PersonID: doctor.PersonId,
		Body: fmt.Sprintf(`Hi %s,

It’s been 8 weeks since your last visit, and I wanted to check in and see how things are going. 

Please complete your follow up visit, and I will review it to see if we need to make any adjustments to your treatment plan. 

Sincerely,
%s`, patient.FirstName, doctor.ShortDisplayName),
	}

	err = messages.CreateMessageAndAttachments(msg, []*messages.Attachment{
		&messages.Attachment{
			Type:  common.AttachmentTypeVisit,
			ID:    followupVisit.PatientVisitId.Int64(),
			Title: "Follow-up Visit",
		},
	}, doctor.PersonId, doctorID, ctxt.Role, f.dataAPI)

	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	people, err := f.dataAPI.GetPeople([]int64{doctor.PersonId})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	person := people[doctor.PersonId]

	f.dispatcher.Publish(&messages.PostEvent{
		Message: msg,
		Case:    patientCase,
		Person:  person,
	})

	apiservice.WriteJSONSuccess(w)
}
