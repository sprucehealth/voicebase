package patient_file

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/mapstructure"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/patient"
)

type doctorPatientVisitReviewHandler struct {
	DataAPI            api.DataAPI
	dispatcher         *dispatch.Dispatcher
	store              storage.Store
	expirationDuration time.Duration
}

func NewDoctorPatientVisitReviewHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, store storage.Store, expirationDuration time.Duration) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(
			&doctorPatientVisitReviewHandler{
				DataAPI:            dataAPI,
				dispatcher:         dispatcher,
				store:              store,
				expirationDuration: expirationDuration,
			}), []string{"GET"})
}

type visitReviewRequestData struct {
	PatientVisitID int64 `schema:"patient_visit_id,required"`
}

type doctorPatientVisitReviewResponse struct {
	Patient            *common.Patient        `json:"patient"`
	PatientVisit       *common.PatientVisit   `json:"patient_visit"`
	PatientVisitReview map[string]interface{} `json:"visit_review"`
}

func (p *doctorPatientVisitReviewHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	doctorID, err := p.DataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	requestData := &visitReviewRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	} else if requestData.PatientVisitID == 0 {
		return false, apiservice.NewValidationError("patient_visit_id must be specified")
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	patientVisit, err := p.DataAPI.GetPatientVisitFromID(requestData.PatientVisitID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientVisit] = patientVisit

	if ctxt.Role == api.DOCTOR_ROLE {
		// update the status of the case and the item in the doctor's queue
		if patientVisit.Status == common.PVStatusRouted {
			pvStatus := common.PVStatusReviewing
			if err := p.DataAPI.UpdatePatientVisit(requestData.PatientVisitID, &api.PatientVisitUpdate{Status: &pvStatus}); err != nil {
				return false, err
			}
			if err := p.DataAPI.MarkPatientVisitAsOngoingInDoctorQueue(doctorID, requestData.PatientVisitID); err != nil {
				return false, err
			}
		}

		p.dispatcher.Publish(&PatientVisitOpenedEvent{
			PatientVisit: patientVisit,
			PatientID:    patientVisit.PatientID.Int64(),
			DoctorID:     doctorID,
			Role:         ctxt.Role,
		})
	}

	// ensure that the doctor is authorized to work on this case
	if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorID,
		patientVisit.PatientID.Int64(), patientVisit.PatientCaseID.Int64(), p.DataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (p *doctorPatientVisitReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patientVisit := ctxt.RequestCache[apiservice.PatientVisit].(*common.PatientVisit)

	renderedLayout, err := VisitReviewLayout(p.DataAPI, p.store, p.expirationDuration, patientVisit, r.Host)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patient, err := p.DataAPI.GetPatientFromID(patientVisit.PatientID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	response := &doctorPatientVisitReviewResponse{}
	response.PatientVisit = patientVisit
	response.Patient = patient
	response.PatientVisitReview = renderedLayout

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, response)
}

func VisitReviewLayout(
	dataAPI api.DataAPI,
	store storage.Store,
	expirationDuration time.Duration,
	visit *common.PatientVisit,
	apiDomain string) (map[string]interface{}, error) {

	visitLayout, err := patient.IntakeLayoutForVisit(dataAPI, store, expirationDuration, visit)
	if err != nil {
		return nil, err
	}

	context, err := buildContext(dataAPI, visitLayout, visit)
	if err != nil {
		return nil, err
	}

	// when rendering the layout for the doctor, ignore views who's keys are missing
	// if we are dealing with a visit that is open, as it is possible that the patient
	// has not answered all questions
	context.IgnoreMissingKeys = (visit.Status == common.PVStatusOpen)

	data, _, err := dataAPI.ReviewLayoutForIntakeLayoutVersionID(visit.LayoutVersionID.Int64(), visit.PathwayID.Int64(), visit.SKU)
	if err != nil {
		return nil, err
	}

	// first we unmarshal the json into a generic map structure
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}

	// then we provide the registry from which to pick out the types of native structures
	// to use when parsing the template into a native go structure
	sectionList := info_intake.DVisitReviewSectionListView{}
	decoderConfig := &mapstructure.DecoderConfig{
		Result:   &sectionList,
		TagName:  "json",
		Registry: *info_intake.DVisitReviewViewTypeRegistry,
	}

	d, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, err
	}

	// assuming that the map structure has the visit_review section here.
	if err := d.Decode(jsonData["visit_review"]); err != nil {
		return nil, err
	}

	return sectionList.Render(context)
}
