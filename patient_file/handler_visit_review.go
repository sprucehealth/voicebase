package patient_file

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/SpruceHealth/mapstructure"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/patient"
	"golang.org/x/net/context"
)

type doctorPatientVisitReviewHandler struct {
	dataAPI            api.DataAPI
	dispatcher         *dispatch.Dispatcher
	mediaStore         *media.Store
	expirationDuration time.Duration
	webDomain          string
}

func NewDoctorPatientVisitReviewHandler(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, mediaStore *media.Store, expirationDuration time.Duration, webDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(
				&doctorPatientVisitReviewHandler{
					dataAPI:            dataAPI,
					dispatcher:         dispatcher,
					mediaStore:         mediaStore,
					expirationDuration: expirationDuration,
					webDomain:          webDomain,
				})),
		httputil.Get)
}

type visitReviewRequestData struct {
	PatientVisitID int64 `schema:"patient_visit_id,required"`
}

type VisitReviewResponse struct {
	Patient            *common.Patient        `json:"patient"`
	PatientVisit       *common.PatientVisit   `json:"patient_visit"`
	PatientVisitReview map[string]interface{} `json:"visit_review"`
}

func (p *doctorPatientVisitReviewHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	doctorID, err := p.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctorID] = doctorID

	requestData := &visitReviewRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	} else if requestData.PatientVisitID == 0 {
		return false, apiservice.NewValidationError("patient_visit_id must be specified")
	}
	requestCache[apiservice.CKRequestData] = requestData

	patientVisit, err := p.dataAPI.GetPatientVisitFromID(requestData.PatientVisitID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPatientVisit] = patientVisit

	if account.Role == api.RoleDoctor {
		// update the status of the case and the item in the doctor's queue
		if patientVisit.Status == common.PVStatusRouted {
			pvStatus := common.PVStatusReviewing
			if _, err := p.dataAPI.UpdatePatientVisit(requestData.PatientVisitID, &api.PatientVisitUpdate{Status: &pvStatus}); err != nil {
				return false, err
			}
			if err := p.dataAPI.MarkPatientVisitAsOngoingInDoctorQueue(doctorID, requestData.PatientVisitID); err != nil {
				return false, err
			}
		}

		p.dispatcher.Publish(&PatientVisitOpenedEvent{
			PatientVisit: patientVisit,
			PatientID:    patientVisit.PatientID,
			DoctorID:     doctorID,
			Role:         account.Role,
		})
	}

	// ensure that the doctor is authorized to work on this case
	if err := apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctorID,
		patientVisit.PatientID, patientVisit.PatientCaseID.Int64(), p.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (p *doctorPatientVisitReviewHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	patientVisit := requestCache[apiservice.CKPatientVisit].(*common.PatientVisit)

	patient, err := p.dataAPI.GetPatientFromID(patientVisit.PatientID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	renderedLayout, err := VisitReviewLayout(p.dataAPI, patient, p.mediaStore, p.expirationDuration, patientVisit, r.Host, p.webDomain)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	response := &VisitReviewResponse{
		PatientVisit:       patientVisit,
		Patient:            patient,
		PatientVisitReview: renderedLayout,
	}
	httputil.JSONResponse(w, http.StatusOK, response)
}

func VisitReviewLayout(
	dataAPI api.DataAPI,
	pat *common.Patient,
	mediaStore *media.Store,
	expirationDuration time.Duration,
	visit *common.PatientVisit,
	apiDomain string,
	webDomain string,
) (map[string]interface{}, error) {
	intakeInfo, err := patient.IntakeLayoutForVisit(dataAPI, apiDomain, webDomain, mediaStore, expirationDuration, visit, pat, api.RoleDoctor)
	if err != nil {
		return nil, err
	}

	context, err := buildContext(dataAPI, mediaStore, expirationDuration, intakeInfo.ClientLayout.InfoIntakeLayout, pat, visit)
	if err != nil {
		return nil, err
	}

	// when rendering the layout for the doctor, ignore views who's keys are missing
	// if we are dealing with a visit that is open, as it is possible that the patient
	// has not answered all questions
	context.IgnoreMissingKeys = (visit.Status == common.PVStatusOpen)

	pathway, err := dataAPI.PathwayForTag(visit.PathwayTag, api.PONone)
	if err != nil {
		return nil, err
	}

	data, _, err := dataAPI.ReviewLayoutForIntakeLayoutVersionID(visit.LayoutVersionID.Int64(), pathway.ID, visit.SKUType)
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
