package patient_file

import (
	"bytes"
	"net/http"
	"text/template"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/messages"
	patientpkg "github.com/sprucehealth/backend/patient"
)

type followupHandler struct {
	dataAPI            api.DataAPI
	authAPI            api.AuthAPI
	dispatcher         *dispatch.Dispatcher
	expirationDuration time.Duration
}

type followupRequestData struct {
	CaseID int64 `json:"case_id,string"`
}

func NewFollowupHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, expirationDuration time.Duration, dispatcher *dispatch.Dispatcher) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(&followupHandler{
				dataAPI:            dataAPI,
				authAPI:            authAPI,
				dispatcher:         dispatcher,
				expirationDuration: expirationDuration,
			})),
		httputil.Post)
}

func (f *followupHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	account := apiservice.MustCtxAccount(ctx)
	requestCache := apiservice.MustCtxCache(ctx)
	if account.Role != api.RoleDoctor && account.Role != api.RoleCC {
		return false, nil
	}

	var rd followupRequestData
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	requestCache[apiservice.CKRequestData] = rd

	doctorID, err := f.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctorID] = doctorID

	patientCase, err := f.dataAPI.GetPatientCaseFromID(rd.CaseID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPatientCase] = patientCase

	if account.Role == api.RoleDoctor {
		if err := apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctorID,
			patientCase.PatientID, patientCase.ID.Int64(), f.dataAPI); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (f *followupHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	patientCase := requestCache[apiservice.CKPatientCase].(*common.PatientCase)
	doctorID := requestCache[apiservice.CKDoctorID].(int64)
	account := apiservice.MustCtxAccount(ctx)

	patient, err := f.dataAPI.GetPatientFromID(patientCase.PatientID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// first create the followup visit
	followupVisit, err := patientpkg.CreatePendingFollowup(patient, patientCase, f.dataAPI, f.authAPI, f.dispatcher)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	personID, err := f.dataAPI.GetPersonIDByRole(account.Role, doctorID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	body, err := bodyOfCaseMessageForFollowup(patientCase.ID.Int64(), patient, f.dataAPI)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
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
			ID:    followupVisit.ID.Int64(),
			Title: "Follow-up Visit",
		},
	},
		personID, doctorID, account.Role, f.dataAPI)

	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	people, err := f.dataAPI.GetPeople([]int64{personID})
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	person := people[personID]

	f.dispatcher.Publish(&messages.PostEvent{
		Message: caseMsg,
		Case:    patientCase,
		Person:  person,
	})

	httputil.JSONResponse(w, http.StatusOK, &struct {
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
		if member.ProviderRole == api.RoleDoctor {
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
