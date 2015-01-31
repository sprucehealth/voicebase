package patient_file

import (
	"bytes"
	"net/http"
	"text/template"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
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
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&followupHandler{
			dataAPI:            dataAPI,
			authAPI:            authAPI,
			dispatcher:         dispatcher,
			expirationDuration: expirationDuration,
			store:              store,
		}), []string{"POST"})
}

func (f *followupHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.DOCTOR_ROLE && ctxt.Role != api.MA_ROLE {
		return false, nil
	}

	var rd followupRequestData
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = rd

	doctorID, err := f.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	patientCase, err := f.dataAPI.GetPatientCaseFromID(rd.CaseID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientCase] = patientCase

	if ctxt.Role == api.DOCTOR_ROLE {
		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorID,
			patientCase.PatientID.Int64(), patientCase.ID.Int64(), f.dataAPI); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (f *followupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patientCase := ctxt.RequestCache[apiservice.PatientCase].(*common.PatientCase)
	doctorID := ctxt.RequestCache[apiservice.DoctorID].(int64)

	patient, err := f.dataAPI.GetPatientFromID(patientCase.PatientID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// first create the followup visit
	followupVisit, err := patientpkg.CreatePendingFollowup(patient, patientCase, f.dataAPI, f.authAPI, f.dispatcher)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	personID, err := f.dataAPI.GetPersonIDByRole(ctxt.Role, doctorID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	body, err := bodyOfCaseMessageForFollowup(patientCase.ID.Int64(), patient, f.dataAPI)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// now create the message
	caseMsg := &common.CaseMessage{
		CaseID:   patientCase.ID.Int64(),
		PersonID: personID,
		Body:     body,
	}

	err = messages.CreateMessageAndAttachments(caseMsg, []*messages.Attachment{
		&messages.Attachment{
			Type:  common.AttachmentTypeVisit,
			ID:    followupVisit.PatientVisitID.Int64(),
			Title: "Follow-up Visit",
		},
	},
		personID, doctorID, ctxt.Role, f.dataAPI)

	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	people, err := f.dataAPI.GetPeople([]int64{personID})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	person := people[personID]

	f.dispatcher.Publish(&messages.PostEvent{
		Message: caseMsg,
		Case:    patientCase,
		Person:  person,
	})

	apiservice.WriteJSON(w, &struct {
		MessageID int64 `json:"message_id,string"`
	}{
		MessageID: caseMsg.ID,
	})
}

type msgContext struct {
	DoctorShortDisplayName string
}

var msg = `Tap the link to begin your follow-up visit with {{.DoctorShortDisplayName}}`

var tmpl *template.Template

func init() {
	var err error
	tmpl, err = template.New("").Parse(msg)
	if err != nil {
		panic(err)
	}
}

func bodyOfCaseMessageForFollowup(patientCaseID int64, patient *common.Patient, dataAPI api.DataAPI) (string, error) {
	var doctorShortDisplayName string

	members, err := dataAPI.GetActiveMembersOfCareTeamForCase(patientCaseID, true)
	if err != nil {
		return "", err
	}

	for _, member := range members {
		if member.ProviderRole == api.DOCTOR_ROLE {
			doctorShortDisplayName = member.ShortDisplayName
		}
	}
	mCtxt := msgContext{
		DoctorShortDisplayName: doctorShortDisplayName,
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, mCtxt); err != nil {
		return "", err
	}

	return b.String(), err
}
