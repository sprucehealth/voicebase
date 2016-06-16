package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type cancelScheduledMessageHandler struct {
	dataAPI    api.DataAPI
	dispatcher dispatch.Publisher
}

type CancelScheduledMessageRequest struct {
	MessageID int64 `json:"message_id,string"`
	Undo      bool  `json:"undo"`
}

func NewCancelScheduledMessageHandler(dataAPI api.DataAPI, dispatcher dispatch.Publisher) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.NoAuthorizationRequired(
					&cancelScheduledMessageHandler{
						dataAPI:    dataAPI,
						dispatcher: dispatcher,
					})),
			api.RoleCC), httputil.Put)
}

func (c *cancelScheduledMessageHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := apiservice.MustCtxAccount(ctx)

	var req CancelScheduledMessageRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	} else if req.MessageID == 0 {
		apiservice.WriteValidationError(ctx, "message_id cannot be 0", w, r)
		return
	}

	tpSchedMsg, err := c.dataAPI.TreatmentPlanScheduledMessage(req.MessageID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if tpSchedMsg.SentTime != nil {
		apiservice.WriteValidationError(ctx, "Message has already been sent so cannot be cancelled or undone.", w, r)
		return
	}

	doctorID, err := c.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	tp, err := c.dataAPI.GetAbridgedTreatmentPlan(tpSchedMsg.TreatmentPlanID, 0)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	} else if tp.Status != common.TPStatusActive {
		apiservice.WriteValidationError(ctx, "Treatment plan is not active so message cannot be cancelled.", w, r)
		return
	}

	cancelled, err := c.dataAPI.CancelTreatmentPlanScheduledMessage(req.MessageID, req.Undo)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if cancelled {
		c.dispatcher.Publish(&TreatmentPlanScheduledMessageCancelledEvent{
			DoctorID:        doctorID,
			TreatmentPlanID: tp.ID.Int64(),
			PatientID:       tp.PatientID,
			CaseID:          tp.PatientCaseID.Int64(),
			Undone:          req.Undo,
		})
	}

	apiservice.WriteJSONSuccess(w)
}
