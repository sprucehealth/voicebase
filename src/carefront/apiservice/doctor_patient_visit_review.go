package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/info_intake"
	"carefront/libs/pharmacy"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/SpruceHealth/mapstructure"
	"github.com/gorilla/schema"
)

type DoctorPatientVisitReviewHandler struct {
	DataApi                    api.DataAPI
	PharmacySearchService      pharmacy.PharmacySearchAPI
	LayoutStorageService       api.CloudStorageAPI
	PatientPhotoStorageService api.CloudStorageAPI
}

type DoctorPatientVisitReviewRequestBody struct {
	PatientVisitId  int64 `schema:"patient_visit_id"`
	TreatmentPlanId int64 `schema:"treatment_plan_id"`
}

type DoctorPatientVisitReviewResponse struct {
	Patient            *common.Patient        `json:"patient"`
	PatientVisit       *common.PatientVisit   `json:"patient_visit"`
	TreatmentPlanId    int64                  `json:"treatment_plan_id"`
	PatientVisitReview map[string]interface{} `json:"visit_review"`
}

func (p *DoctorPatientVisitReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData DoctorPatientVisitReviewRequestBody
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisitId := requestData.PatientVisitId
	treatmentPlanId := requestData.TreatmentPlanId
	if err := ensureTreatmentPlanOrPatientVisitIdPresent(p.DataApi, treatmentPlanId, &patientVisitId); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisit, err := p.DataApi.GetPatientVisitFromId(patientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient visit information from database based on provided patient visit id : "+err.Error())
		return
	}

	// ensure that the doctor is authorized to work on this case
	patientVisitReviewData, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisit.PatientVisitId.Int64(), GetContext(r).AccountId, p.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	// udpate the status of the case and the item in the doctor's queue
	if patientVisit.Status == api.CASE_STATUS_SUBMITTED {
		treatmentPlanId, err = p.DataApi.StartNewTreatmentPlanForPatientVisit(patientVisit.PatientId.Int64(), patientVisit.PatientVisitId.Int64(), patientVisitReviewData.DoctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the visit to reviewing: "+err.Error())
			return
		}

		if err := p.DataApi.MarkPatientVisitAsOngoingInDoctorQueue(patientVisitReviewData.DoctorId, patientVisit.PatientVisitId.Int64()); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the item in the queue for the doctor that speaks to this patient visit: "+err.Error())
			return
		}

		if err := p.DataApi.RecordDoctorAssignmentToPatientVisit(patientVisit.PatientVisitId.Int64(), patientVisitReviewData.DoctorId); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to assign the patient visit to this doctor: "+err.Error())
			return
		}
	} else {
		treatmentPlanId, err = p.DataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, patientVisit.PatientVisitId.Int64())
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan id for patient visit: "+err.Error())
			return
		}
	}

	patientVisitLayout, _, err := getClientLayoutForPatientVisit(patientVisitId, api.EN_LANGUAGE_ID, p.DataApi, p.LayoutStorageService)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient visit layout: "+err.Error())
		return
	}

	// get all questions presented to the patient in the patient visit layout
	questions := getQuestionsInPatientVisitLayout(patientVisitLayout)
	questionIds := getQuestionIdsInPatientVisitLayout(patientVisitLayout)

	// get all the answers the patient entered for the questions (note that there may not be an answer for every question)
	patientAnswersForQuestions, err := p.DataApi.GetPatientAnswersForQuestionsBasedOnQuestionIds(questionIds, patientVisit.PatientId.Int64(), patientVisit.PatientVisitId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient answers for questions : "+err.Error())
		return
	}

	context, err := populateContextForRenderingLayout(patientAnswersForQuestions, questions, p.DataApi, p.PatientPhotoStorageService)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to populate context for rendering layout: "+err.Error())
		return
	}

	data, err := p.getLatestDoctorVisitReviewLayout(patientVisit)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get visit review template for doctor")
	}

	// first we unmarshal the json into a generic map structure
	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unbale to unmarshal file contents into map[string]interface{}: "+err.Error())
	}

	// then we provide the registry from which to pick out the types of native structures
	// to use when parsing the template into a native go structure
	sectionList := info_intake.DVisitReviewSectionListView{}
	decoderConfig := &mapstructure.DecoderConfig{
		Result:  &sectionList,
		TagName: "json",
	}
	decoderConfig.SetRegistry(info_intake.DVisitReviewViewTypeRegistry.Map())

	d, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create new decoder: "+err.Error())
		return
	}

	// assuming that the map structure has the visit_review section here.
	err = d.Decode(jsonData["visit_review"])
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse template into structure: "+err.Error())
		return
	}

	renderedJsonData, err := sectionList.Render(context)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to render template into expected view layout for doctor visit review: "+err.Error())
		return
	}

	response := &DoctorPatientVisitReviewResponse{}
	response.PatientVisit = patientVisit
	patient, err := p.DataApi.GetPatientFromId(patientVisit.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on id: "+err.Error())
		return
	}

	response.Patient = patient
	response.TreatmentPlanId = treatmentPlanId
	response.PatientVisitReview = renderedJsonData

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, response)
}

func (d *DoctorPatientVisitReviewHandler) getLatestDoctorVisitReviewLayout(patientVisit *common.PatientVisit) ([]byte, error) {
	bucket, key, region, _, err := d.DataApi.GetStorageInfoOfCurrentActiveDoctorLayout(patientVisit.HealthConditionId.Int64())
	if err != nil {
		return nil, err
	}

	data, _, err := d.LayoutStorageService.GetObjectAtLocation(bucket, key, region)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func populateContextForRenderingLayout(patientAnswersForQuestions map[int64][]*common.AnswerIntake, questions []*info_intake.Question, dataApi api.DataAPI, photoStorageService api.CloudStorageAPI) (common.ViewContext, error) {
	context := common.NewViewContext()

	for _, contextPopulator := range genericPopulators {
		if err := contextPopulator.PopulateViewContextWithInfo(patientAnswersForQuestions, questions, context, dataApi); err != nil {
			return nil, err
		}
	}

	// go through each question
	for _, question := range questions {
		contextPopulator, ok := patientQAPopulators[question.QuestionTypes[0]]
		if !ok {
			return nil, fmt.Errorf("Context populator not found for question with type %s", question.QuestionTypes[0])
		}

		if err := contextPopulator.PopulateViewContextWithPatientQA(patientAnswersForQuestions[question.QuestionId], question, context, dataApi, photoStorageService); err != nil {
			return nil, err
		}
	}

	return *context, nil
}
