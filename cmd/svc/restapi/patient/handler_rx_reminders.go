package patient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/app_url"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/erx"
	"github.com/sprucehealth/backend/cmd/svc/restapi/responses"
	"github.com/sprucehealth/backend/cmd/svc/restapi/treatment_plan"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/views"
	"github.com/sprucehealth/schema"
)

// rxReminderService defines the methods used by this handler
type rxReminderService interface {
	RemindersForPatient(patientID common.PatientID) (map[int64]*common.RXReminder, error)
	DeleteRXReminder(treatmentID int64) error
	CreateRXReminder(reminder *common.RXReminder) error
	UpdateRXReminder(treatmentID int64, reminder *common.RXReminder) error
}

// treatmentAccessValidator defines the validation mechanisms required by this handler
type treatmentService interface {
	PatientCanAccessTreatment(patientID common.PatientID, treatmentID int64) (bool, error)
	TreatmentsForPatient(patientID common.PatientID) ([]*common.Treatment, error)
}

// drugService exposes the methods required by this handler
type drugService interface {
	MultiQueryDrugDetailIDs(queries []*api.DrugDetailsQuery) ([]int64, error)
}

type rxRemindersHandler struct {
	rxReminderSvc rxReminderService
	treatmentSvc  treatmentService
	drugSvc       drugService
}

// rxRemindersDELETERequest represents the data expected to be associated with a successful DELETE request
type rxRemindersDELETERequest struct {
	TreatmentID int64 `schema:"treatment_id,required"`
}

// rxRemindersGETRequest represents the data expected to associated with a successful GET request
type rxRemindersGETRequest struct {
	IncludeViews bool `schema:"include_views"`
}

// rxRemindersGETResponse represents the data expected to returned from a successful GET request
type rxRemindersGETResponse struct {
	TreatmentViews map[string]views.View            `json:"views,omitempty"`
	Reminders      map[string]*responses.RXReminder `json:"reminders"`
}

// rxRemindersPOSTRequest represents the data expected to be associated with a successful POST request
type rxRemindersPOSTRequest struct {
	TreatmentID  int64  `json:"treatment_id,string"`
	ReminderText string `json:"reminder_text"`
	Interval     string `json:"interval"`
	RXRInterval  common.RXRInterval
	Days         []string `json:"days"`
	RXRDays      []common.RXRDay
	Times        []string `json:"times"`
}

// NewRXReminderHandlerHandler returns an initialized instance of rxRemindersHandler
func NewRXReminderHandlerHandler(rxReminderSvc rxReminderService, treatmentSvc treatmentService, drugSvc drugService) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(&rxRemindersHandler{
					rxReminderSvc: rxReminderSvc,
					treatmentSvc:  treatmentSvc,
					drugSvc:       drugSvc,
				})), api.RolePatient),
		httputil.Delete, httputil.Get, httputil.Post, httputil.Put)
}

func (h *rxRemindersHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	switch r.Method {
	case httputil.Delete:
		rd, err := h.parseDELETERequest(ctx, r)
		if err != nil {
			return false, apiservice.NewValidationError(err.Error())
		}
		access, err := h.validateDELETEAccess(ctx, rd)
		if err != nil {
			return false, err
		} else if !access {
			return false, nil
		}
		requestCache[apiservice.CKRequestData] = rd
		return access, nil
	case httputil.Post:
		rd, err := h.parsePUTPOSTRequest(ctx, r)
		if err != nil {
			return false, apiservice.NewValidationError(err.Error())
		}
		access, err := h.validatePOSTAccess(ctx, rd)
		if err != nil {
			return false, err
		} else if !access {
			return false, nil
		}
		requestCache[apiservice.CKRequestData] = rd
		return access, nil
	case httputil.Get:
		return true, nil
	case httputil.Put:
		rd, err := h.parsePUTPOSTRequest(ctx, r)
		if err != nil {
			return false, apiservice.NewValidationError(err.Error())
		}
		access, err := h.validatePUTAccess(ctx, rd)
		if err != nil {
			return false, err
		} else if !access {
			return false, nil
		}
		requestCache[apiservice.CKRequestData] = rd
		return access, nil
	}
	return true, nil
}

func (h *rxRemindersHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case httputil.Delete:
		h.serveDELETE(ctx, w, r, apiservice.MustCtxCache(ctx)[apiservice.CKRequestData].(*rxRemindersDELETERequest))
	case httputil.Get:
		rd, err := h.parseGETRequest(ctx, r)
		if err != nil {
			apiservice.WriteValidationError(ctx, err.Error(), w, r)
			return
		}
		h.serveGET(ctx, w, r, rd)
	case httputil.Post:
		h.servePOST(ctx, w, r, apiservice.MustCtxCache(ctx)[apiservice.CKRequestData].(*rxRemindersPOSTRequest))
	case httputil.Put:
		h.servePUT(ctx, w, r, apiservice.MustCtxCache(ctx)[apiservice.CKRequestData].(*rxRemindersPOSTRequest))
	}
}

func (h *rxRemindersHandler) parseDELETERequest(ctx context.Context, r *http.Request) (*rxRemindersDELETERequest, error) {
	rd := &rxRemindersDELETERequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *rxRemindersHandler) validateDELETEAccess(ctx context.Context, rd *rxRemindersDELETERequest) (bool, error) {
	access, err := h.validateTreatmentAccess(ctx, rd.TreatmentID)
	return access, errors.Trace(err)
}

func (h *rxRemindersHandler) serveDELETE(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *rxRemindersDELETERequest) {
	if err := h.rxReminderSvc.DeleteRXReminder(rd.TreatmentID); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	apiservice.WriteJSONSuccess(w)
}

func (h *rxRemindersHandler) parseGETRequest(ctx context.Context, r *http.Request) (*rxRemindersGETRequest, error) {
	rd := &rxRemindersGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *rxRemindersHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *rxRemindersGETRequest) {
	var resp rxRemindersGETResponse
	parallel := conc.NewParallel()
	parallel.Go(func() error {
		reminders, err := h.rxReminderSvc.RemindersForPatient(apiservice.MustCtxPatient(ctx).ID)
		if err != nil {
			return err
		}

		respReminders := make(map[string]*responses.RXReminder, len(reminders))
		for k, r := range reminders {
			respReminders[strconv.FormatInt(k, 10)] = responses.TransformRXReminder(r)
		}
		resp.Reminders = respReminders
		return nil
	})

	if rd.IncludeViews {
		parallel.Go(func() error {
			var err error
			treatments, err := h.treatmentSvc.TreatmentsForPatient(apiservice.MustCtxPatient(ctx).ID)
			if err != nil {
				return err
			}
			resp.TreatmentViews, err = h.viewsForTreatments(treatments)
			if err != nil {
				return err
			}
			return nil
		})
	}
	if err := parallel.Wait(); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, resp)
}

func (h *rxRemindersHandler) parsePUTPOSTRequest(ctx context.Context, r *http.Request) (*rxRemindersPOSTRequest, error) {
	var err error
	rd := &rxRemindersPOSTRequest{}
	if err = json.NewDecoder(r.Body).Decode(rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.TreatmentID == 0 || rd.ReminderText == "" || len(rd.Times) == 0 || rd.Interval == "" {
		return nil, errors.New("treatment_id, reminder_test, times, interval required")
	}

	rd.RXRInterval, err = common.ParseRXRInterval(rd.Interval)
	if err != nil {
		return nil, err
	}

	if rd.RXRInterval == common.RXRIntervalCustom && len(rd.Days) == 0 {
		return nil, errors.New("days required for CUSTOM interval")
	}

	for _, d := range rd.Days {
		rxrDay, err := common.ParseRXRDay(d)
		if err != nil {
			return nil, err
		}
		rd.RXRDays = append(rd.RXRDays, rxrDay)
	}

	for _, t := range rd.Times {
		if _, err := common.ParseRXRTime(t); err != nil {
			return nil, err
		}
	}
	return rd, nil
}

func (h *rxRemindersHandler) validatePOSTAccess(ctx context.Context, rd *rxRemindersPOSTRequest) (bool, error) {
	access, err := h.validateTreatmentAccess(ctx, rd.TreatmentID)
	return access, errors.Trace(err)
}

func (h *rxRemindersHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *rxRemindersPOSTRequest) {
	rxReminder := &common.RXReminder{
		TreatmentID:  rd.TreatmentID,
		ReminderText: rd.ReminderText,
		Interval:     rd.RXRInterval,
		Days:         rd.RXRDays,
		Times:        strings.Join(rd.Times, `,`),
	}
	if err := h.rxReminderSvc.CreateRXReminder(rxReminder); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	// A successful post should refresh the reminder list
	h.serveGET(ctx, w, r, &rxRemindersGETRequest{IncludeViews: false})
}

func (h *rxRemindersHandler) validatePUTAccess(ctx context.Context, rd *rxRemindersPOSTRequest) (bool, error) {
	access, err := h.validateTreatmentAccess(ctx, rd.TreatmentID)
	return access, errors.Trace(err)
}

func (h *rxRemindersHandler) servePUT(ctx context.Context, w http.ResponseWriter, r *http.Request, rd *rxRemindersPOSTRequest) {
	rxReminder := &common.RXReminder{
		TreatmentID:  rd.TreatmentID,
		ReminderText: rd.ReminderText,
		Interval:     rd.RXRInterval,
		Days:         rd.RXRDays,
		Times:        strings.Join(rd.Times, `,`),
	}
	if err := h.rxReminderSvc.UpdateRXReminder(rd.TreatmentID, rxReminder); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	// A successful put should refresh the reminder list
	h.serveGET(ctx, w, r, &rxRemindersGETRequest{IncludeViews: false})
}

func (h *rxRemindersHandler) validateTreatmentAccess(ctx context.Context, treatmentID int64) (bool, error) {
	access, err := h.treatmentSvc.PatientCanAccessTreatment(apiservice.MustCtxPatient(ctx).ID, treatmentID)
	return access, errors.Trace(err)
}

func (h *rxRemindersHandler) viewsForTreatments(treatments []*common.Treatment) (map[string]views.View, error) {
	tViews := make(map[string]views.View, len(treatments))
	if len(treatments) == 0 {
		return tViews, nil
	}

	drugQueries := make([]*api.DrugDetailsQuery, len(treatments))
	for i, t := range treatments {
		drugQueries[i] = &api.DrugDetailsQuery{
			NDC:         t.DrugDBIDs[erx.NDC],
			GenericName: t.GenericDrugName,
			Route:       t.DrugRoute,
			Form:        t.DrugForm,
		}
	}
	drugDetails, err := h.drugSvc.MultiQueryDrugDetailIDs(drugQueries)
	if err != nil {
		// It's possible to continue. We just won't return treatment guide buttons
		golog.Errorf("Failed to query for drug details: %s", err.Error())
		// The drugDetails slice is expected to have the same number of elements as treatments
		drugDetails = make([]int64, len(treatments))
	}
	for i, treatment := range treatments {
		iconURL := app_url.IconRXLarge
		if treatment.OTC {
			iconURL = app_url.IconOTCLarge
		}

		var subtitle string
		if treatment.OTC {
			subtitle = "Over-the-counter"
		} else {
			switch treatment.DrugRoute {
			case "topical":
				subtitle = "Topical Prescription"
			case "oral":
				subtitle = "Oral Prescription"
			default:
				subtitle = "Prescription"
			}
		}

		var buttons []views.View
		if drugDetails[i] != 0 {
			buttons = append(buttons, treatment_plan.NewPrescriptionButtonView("Prescription Guide", app_url.IconRXGuide, app_url.ViewTreatmentGuideAction(treatment.ID.Int64())))
		}
		pView := treatment_plan.NewPrescriptionView(treatment, subtitle, iconURL, buttons)
		if err := pView.Validate("treatment"); err != nil {
			return nil, errors.Trace(err)
		}
		tViews[strconv.FormatInt(treatment.ID.Int64(), 10)] = pView
	}

	return tViews, nil
}
