package patient_case

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type notificationsListHandler struct {
	dataAPI   api.DataAPI
	apiDomain string
}

type notificationsListRequestData struct {
	PatientCaseID int64 `schema:"case_id"`
}

type notificationsListResponseData struct {
	Items []common.ClientView `json:"items"`
}

func NewNotificationsListHandler(dataAPI api.DataAPI, apiDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&notificationsListHandler{
				dataAPI:   dataAPI,
				apiDomain: apiDomain,
			}), api.RolePatient), httputil.Get)
}

func (n *notificationsListHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestData := &notificationsListRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	} else if requestData.PatientCaseID == 0 {
		apiservice.WriteValidationError(ctx, "case_id must be specified", w, r)
		return
	}

	patientCase, err := n.dataAPI.GetPatientCaseFromID(requestData.PatientCaseID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	assignments, err := n.dataAPI.GetActiveMembersOfCareTeamForCase(patientCase.ID.Int64(), true)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	notificationItems, err := n.dataAPI.GetNotificationsForCase(requestData.PatientCaseID, NotifyTypes)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	nViewItems := make([]common.ClientView, 0, len(notificationItems))
	for _, notificationItem := range notificationItems {
		nDataItem := notificationItem.Data.(notification)
		if nDataItem.canRenderCaseNotificationView() {
			viewItem, err := nDataItem.makeCaseNotificationView(&caseData{
				Notification:    notificationItem,
				Case:            patientCase,
				CareTeamMembers: assignments,
			})
			if err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}

			nViewItems = append(nViewItems, viewItem)
		}
	}

	httputil.JSONResponse(w, http.StatusOK, &notificationsListResponseData{Items: nViewItems})
}
