package patient_visit

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
)

const (
	zipcodeTag = "<zipcode>"
)

type presubmissionTriageHandler struct {
	dataAPI    api.DataAPI
	dispatcher *dispatch.Dispatcher
}

type presubmissionTriageRequest struct {
	PatientVisitID int64  `json:"patient_visit_id,string"`
	Title          string `json:"title"`
	ActionMessage  string `json:"action_message"`
	ActionURL      string `json:"action_url"`
	Abandon        bool   `json:"abandon"`
}

func NewPreSubmissionTriageHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&presubmissionTriageHandler{
					dataAPI:    dataAPI,
					dispatcher: dispatcher,
				}), api.RolePatient), httputil.Put)
}

func (p *presubmissionTriageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rd presubmissionTriageRequest
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// ensure that the visit is either in an open state or a pre-submission triaged state
	visit, err := p.dataAPI.GetPatientVisitFromID(rd.PatientVisitID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if !(visit.Status == common.PVStatusPreSubmissionTriage || visit.Status == common.PVStatusOpen) {
		apiservice.WriteValidationError("only an open visit can under pre-submission triage", w, r)
		return
	}

	par := conc.NewParallel()
	par.Go(func() error {
		// update the patient visit status
		updatedStatus := common.PVStatusPreSubmissionTriage
		_, err := p.dataAPI.UpdatePatientVisit(rd.PatientVisitID, &api.PatientVisitUpdate{
			ClosedDate: ptr.Time(time.Now()),
			Status:     &updatedStatus,
		})
		return err
	})
	par.Go(func() error {
		updatedStatus := common.PCStatusPreSubmissionTriage
		var timeoutDate *time.Time
		now := time.Now()
		if rd.Abandon {
			updatedStatus = common.PCStatusPreSubmissionTriageDeleted
		} else {
			td := time.Now().Add(24 * time.Hour)
			timeoutDate = &td
		}

		if err := p.dataAPI.UpdatePatientCase(visit.PatientCaseID.Int64(), &api.PatientCaseUpdate{
			Status:     &updatedStatus,
			ClosedDate: &now,
			TimeoutDate: api.NullableTime{
				Valid: true,
				Time:  timeoutDate,
			},
		}); err != nil {
			return err
		}

		title := rd.Title
		if title == "" {
			patientCase, err := p.dataAPI.GetPatientCaseFromID(visit.PatientCaseID.Int64())
			if err != nil {
				return err
			}

			title = fmt.Sprintf("Your %s visit has ended and you should seek medical care today.", strings.ToLower(patientCase.Name))
		}

		actionMessage := rd.ActionMessage
		if actionMessage == "" {
			actionMessage = "How to find a local care provider"
		}

		zipcode, _, err := p.dataAPI.PatientLocation(visit.PatientID.Int64())
		if err != nil {
			return err
		}

		actionURL := rd.ActionURL
		if actionURL == "" {
			actionURL = fmt.Sprintf("https://www.google.com/?gws_rd=ssl#q=urgent+care+in+%s", zipcode)
		} else {
			actionURL = strings.Replace(actionURL, zipcodeTag, zipcode, -1)
		}

		p.dispatcher.Publish(&PreSubmissionVisitTriageEvent{
			VisitID:       visit.ID.Int64(),
			CaseID:        visit.PatientCaseID.Int64(),
			Title:         title,
			ActionMessage: actionMessage,
			ActionURL:     actionURL,
		})

		return nil
	})
	if err := par.Wait(); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSONSuccess(w)
}
