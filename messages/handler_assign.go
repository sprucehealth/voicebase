package messages

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type assignHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

func NewAssignHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(
					&assignHandler{
						dataAPI:    dataAPI,
						dispatcher: dispatcher,
					})),
			api.RoleDoctor, api.RoleCC),
		httputil.Post)
}

func (a *assignHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	requestData := &PostMessageRequest{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	requestCache[apiservice.CKRequestData] = requestData

	doctor, err := a.dataAPI.GetDoctorFromAccountID(account.ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctor] = doctor

	patientCase, err := a.dataAPI.GetPatientCaseFromID(requestData.CaseID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPatientCase] = patientCase

	personID, doctorID, err := validateAccess(a.dataAPI, r, apiservice.MustCtxAccount(ctx), patientCase)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPersonID] = personID
	requestCache[apiservice.CKDoctorID] = doctorID

	return true, nil
}

func (a *assignHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	requestData := requestCache[apiservice.CKRequestData].(*PostMessageRequest)
	patientCase := requestCache[apiservice.CKPatientCase].(*common.PatientCase)
	personID := requestCache[apiservice.CKPersonID].(int64)
	doctorID := requestCache[apiservice.CKDoctorID].(int64)

	account := apiservice.MustCtxAccount(ctx)

	if requestData.CaseID == 0 {
		apiservice.WriteValidationError(ctx, "case_id is required", w, r)
		return
	}

	// MA can only assign a case that is already claimed
	if account.Role == api.RoleCC && !patientCase.Claimed {
		apiservice.WriteValidationError(ctx, "Care coordinator cannot assign a case to a doctor for a case that is not currently claimed by a doctor", w, r)
		return
	}

	people, err := a.dataAPI.GetPeople([]int64{personID})
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	person := people[personID]

	var longDisplayName string
	var doctor *common.Doctor
	var ma *common.Doctor
	switch account.Role {
	case api.RoleCC:
		ma = requestCache[apiservice.CKDoctor].(*common.Doctor)

		// identify the doctor for the case
		assignments, err := a.dataAPI.GetDoctorsAssignedToPatientCase(patientCase.ID.Int64())
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		for _, doctorAssignment := range assignments {
			if doctorAssignment.Status == api.StatusActive {
				doctor, err = a.dataAPI.GetDoctorFromID(doctorAssignment.ProviderID)
				if err != nil {
					apiservice.WriteError(ctx, err, w, r)
					return
				}
				longDisplayName = doctor.LongDisplayName
				break
			}
		}
	case api.RoleDoctor:
		doctor = requestCache[apiservice.CKDoctor].(*common.Doctor)

		careTeam, err := a.dataAPI.GetActiveMembersOfCareTeamForCase(patientCase.ID.Int64(), false)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
		for _, cp := range careTeam {
			if cp.ProviderRole == api.RoleCC {
				ma, err = a.dataAPI.Doctor(cp.ProviderID, true)
				if err != nil {
					apiservice.WriteError(ctx, err, w, r)
					return
				}
				break
			}
		}

		if ma == nil {
			apiservice.WriteError(ctx, fmt.Errorf("No CC assigned to case %d", patientCase.ID.Int64()), w, r)
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

	if err := CreateMessageAndAttachments(msg, requestData.Attachments, personID, doctorID, account.Role, a.dataAPI); err != nil {
		apiservice.WriteError(ctx, err, w, r)
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
