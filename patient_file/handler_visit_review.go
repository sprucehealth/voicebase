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
	return apiservice.SupportedMethods(&doctorPatientVisitReviewHandler{
		DataApi: dataApi,
	}, []string{apiservice.HTTP_GET})
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
	ctxt := apiservice.GetContext(r)

	doctorId, err := p.DataApi.GetDoctorIdFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorId] = doctorId

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
	requestData := ctxt.RequestCache[apiservice.RequestData].(*visitReviewRequestData)
	patientVisit := ctxt.RequestCache[apiservice.PatientVisit].(*common.PatientVisit)

	patientVisitLayout, _, err := apiservice.GetPatientLayoutForPatientVisit(requestData.PatientVisitId, api.EN_LANGUAGE_ID, p.DataApi)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient visit layout: "+err.Error())
		return
	}

	context, err := buildContext(p.DataApi, patientVisitLayout, patientVisit.PatientId.Int64(), requestData.PatientVisitId, r)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	data, err := p.getLatestDoctorVisitReviewLayout(patientVisit)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get visit review template for doctor: "+err.Error())
		return
	}

	// first we unmarshal the json into a generic map structure
	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unbale to unmarshal file contents into map[string]interface{}: "+err.Error())
		return
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
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create new decoder: "+err.Error())
		return
	}

	// assuming that the map structure has the visit_review section here.
	err = d.Decode(jsonData["visit_review"])
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse template into structure: "+err.Error())
		return
	}

	renderedJsonData, err := sectionList.Render(context)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to render template into expected view layout for doctor visit review: "+err.Error())
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
	response.PatientVisitReview = renderedJsonData

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, response)
}

func (d *doctorPatientVisitReviewHandler) getLatestDoctorVisitReviewLayout(patientVisit *common.PatientVisit) ([]byte, error) {
	data, _, err := d.DataApi.GetCurrentActiveDoctorLayout(patientVisit.HealthConditionId.Int64())
	if err != nil {
		return nil, err
	}

	return data, nil
}
