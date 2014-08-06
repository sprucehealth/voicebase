package patient_file

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/mapstructure"
)

type doctorPatientVisitReviewHandler struct {
	DataApi api.DataAPI
}

func NewDoctorPatientVisitReviewHandler(dataApi api.DataAPI) http.Handler {
	return &doctorPatientVisitReviewHandler{
		DataApi: dataApi,
	}
}

type visitReviewRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id,required"`
}

type doctorPatientVisitReviewResponse struct {
	Patient            *common.Patient        `json:"patient"`
	PatientVisit       *common.PatientVisit   `json:"patient_visit"`
	PatientVisitReview map[string]interface{} `json:"visit_review"`
}

func (p *doctorPatientVisitReviewHandler) IsAuthorized(r *http.Request) (bool, error) {
	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewResourceNotFoundError("", r)
	}

	ctxt := apiservice.GetContext(r)

	doctorId, err := p.DataApi.GetDoctorIdFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorId

	requestData := &visitReviewRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	} else if requestData.PatientVisitId == 0 {
		return false, apiservice.NewValidationError("patient_visit_id must be specified", r)
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	patientVisit, err := p.DataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientVisit] = patientVisit

	// udpate the status of the case and the item in the doctor's queue
	if patientVisit.Status == common.PVStatusSubmitted {
		if err := p.DataApi.UpdatePatientVisitStatus(requestData.PatientVisitId, "", common.PVStatusReviewing); err != nil {
			return false, err
		}
		if err := p.DataApi.MarkPatientVisitAsOngoingInDoctorQueue(doctorId, requestData.PatientVisitId); err != nil {
			return false, err
		}
	}

	dispatch.Default.Publish(&PatientVisitOpenedEvent{
		PatientVisit: patientVisit,
		PatientId:    patientVisit.PatientId.Int64(),
		DoctorId:     doctorId,
		Role:         ctxt.Role,
	})

	// ensure that the doctor is authorized to work on this case
	if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctorId,
		patientVisit.PatientId.Int64(), patientVisit.PatientCaseId.Int64(), p.DataApi); err != nil {
		return false, err
	}

	return true, nil
}

func (p *doctorPatientVisitReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patientVisit := ctxt.RequestCache[apiservice.PatientVisit].(*common.PatientVisit)

	renderedLayout, err := VisitReviewLayout(p.DataApi, patientVisit, r.Host)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	patient, err := p.DataApi.GetPatientFromId(patientVisit.PatientId.Int64())
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

func VisitReviewLayout(dataAPI api.DataAPI, visit *common.PatientVisit, apiDomain string) (map[string]interface{}, error) {
	patientVisitLayout, _, err := apiservice.GetPatientLayoutForPatientVisit(visit.PatientVisitId.Int64(), api.EN_LANGUAGE_ID, dataAPI)
	if err != nil {
		return nil, err
	}

	context, err := buildContext(dataAPI, patientVisitLayout, visit.PatientId.Int64(), visit.PatientVisitId.Int64(), apiDomain)
	if err != nil {
		return nil, err
	}

	data, _, err := dataAPI.GetCurrentActiveDoctorLayout(visit.HealthConditionId.Int64())
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
