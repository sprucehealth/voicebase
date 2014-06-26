package patient_file

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/mapstructure"
)

type doctorPatientVisitReviewHandler struct {
	DataApi api.DataAPI
}

func NewDoctorPatientVisitReviewHandler(dataApi api.DataAPI) *doctorPatientVisitReviewHandler {
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

func (p *doctorPatientVisitReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

	var requestData visitReviewRequestData
	if err := apiservice.DecodeRequestData(&requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	} else if requestData.PatientVisitId == 0 {
		apiservice.WriteValidationError("patient_visit_id must be specified", w, r)
	}

	patientVisit, err := p.DataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient visit information from database based on provided patient visit id : "+err.Error())
		return
	}

	patient, err := p.DataApi.GetPatientFromId(patientVisit.PatientId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on id: "+err.Error())
		return
	}

	doctorId, err := p.DataApi.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// udpate the status of the case and the item in the doctor's queue
	if patientVisit.Status == common.PVStatusSubmitted {
		if err := p.DataApi.UpdatePatientVisitStatus(requestData.PatientVisitId, "", common.PVStatusReviewing); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		if err := p.DataApi.MarkPatientVisitAsOngoingInDoctorQueue(doctorId, requestData.PatientVisitId); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	dispatch.Default.Publish(&PatientVisitOpenedEvent{
		PatientVisit: patientVisit,
		PatientId:    patient.PatientId.Int64(),
		DoctorId:     doctorId,
	})

	// ensure that the doctor is authorized to work on this case
	if err := apiservice.ValidateReadAccessToPatientCase(doctorId, patientVisit.PatientId.Int64(), patientVisit.PatientCaseId.Int64(), p.DataApi); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

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

func populateContextForRenderingLayout(patientAnswersForQuestions map[int64][]common.Answer, questions []*info_intake.Question, dataApi api.DataAPI, r *http.Request) (common.ViewContext, error) {
	context := common.NewViewContext()

	for _, contextPopulator := range genericPopulators {
		if err := contextPopulator.populateViewContextWithInfo(patientAnswersForQuestions, questions, context, dataApi); err != nil {
			return nil, err
		}
	}

	// go through each question
	for _, question := range questions {
		contextPopulator, ok := patientQAPopulators[question.QuestionType]
		if !ok {
			return nil, fmt.Errorf("Context populator not found for question with type %s", question.QuestionType)
		}

		if err := contextPopulator.populateViewContextWithPatientQA(patientAnswersForQuestions[question.QuestionId], question, context, dataApi, r); err != nil {
			return nil, err
		}
	}

	return *context, nil
}
