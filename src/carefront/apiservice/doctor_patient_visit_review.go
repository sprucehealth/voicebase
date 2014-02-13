package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/info_intake"
	"carefront/libs/pharmacy"
	"encoding/json"
	"github.com/gorilla/schema"
	"net/http"
	"strings"
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
	DoctorLayout *info_intake.PatientVisitOverview `json:"patient_visit_overview,omitempty"`
}

func NewDoctorPatientVisitReviewHandler(dataApi api.DataAPI, layoutStorageService api.CloudStorageAPI, patientPhotoStorageService api.CloudStorageAPI) *DoctorPatientVisitReviewHandler {
	return &DoctorPatientVisitReviewHandler{DataApi: dataApi, LayoutStorageService: layoutStorageService, PatientPhotoStorageService: patientPhotoStorageService}
}

func (p *DoctorPatientVisitReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(DoctorPatientVisitReviewRequestBody)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisitId := requestData.PatientVisitId
	treatmentPlanId := requestData.TreatmentPlanId
	err = ensureTreatmentPlanOrPatientVisitIdPresent(p.DataApi, treatmentPlanId, &patientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisit, err := p.DataApi.GetPatientVisitFromId(patientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient visit information from database based on provided patient visit id : "+err.Error())
		return
	}

	// ensure that the doctor is authorized to work on this case
	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisit.PatientVisitId, GetContext(r).AccountId, p.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	// udpate the status of the case and the item in the doctor's queue
	if patientVisit.Status == api.CASE_STATUS_SUBMITTED {
		_, err = p.DataApi.StartNewTreatmentPlanForPatientVisit(patientVisit.PatientId, patientVisit.PatientVisitId, doctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the visit to reviewing: "+err.Error())
			return
		}

		err = p.DataApi.MarkPatientVisitAsOngoingInDoctorQueue(doctorId, patientVisit.PatientVisitId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the item in the queue for the doctor that speaks to this patient visit: "+err.Error())
			return
		}

		err = p.DataApi.RecordDoctorAssignmentToPatientVisit(patientVisit.PatientVisitId, doctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to assign the patient visit to this doctor: "+err.Error())
			return
		}
	}

	patient, err := p.DataApi.GetPatientFromId(patientVisit.PatientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient info based on account id: "+err.Error())
		return
	}

	pharmacySelection, err := p.DataApi.GetPatientPharmacySelection(patient.PatientId)
	if err != nil && err != api.NoRowsError {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to gte patient's pharmacy selection: "+err.Error())
		return
	}

	if pharmacySelection != nil && pharmacySelection.Id != "" && pharmacySelection.Address == "" {
		patientPharmacy, err := p.PharmacySearchService.GetPharmacyBasedOnId(pharmacySelection.Id)
		if err != nil && err != pharmacy.NoPharmacyExists {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get pharmacy based on id: "+err.Error())
			return
		}
		patient.Pharmacy = patientPharmacy
	} else {
		patient.Pharmacy = pharmacySelection
	}

	bucket, key, region, _, err := p.DataApi.GetStorageInfoOfCurrentActiveDoctorLayout(patientVisit.HealthConditionId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the active layout version for the doctor's view of the patient visit: "+err.Error())
		return
	}

	data, _, err := p.LayoutStorageService.GetObjectAtLocation(bucket, key, region)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor layout for patient visit from s3: "+err.Error())
		return
	}

	patientVisitOverview := &info_intake.PatientVisitOverview{}
	err = json.Unmarshal(data, patientVisitOverview)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse doctor layout for patient visit from s3: "+err.Error())
		return
	}

	fillInPatientVisitInfoIntoOverview(patientVisit, patientVisitOverview)
	patientVisitOverview.Patient = patient

	// capitalizing the gender for display purposes. TODO Make how we do this better for v1
	patientVisitOverview.Patient.Gender = strings.Title(patient.Gender)

	p.filterOutGenderSpecificQuestionsAndSubSectionsFromOverview(patientVisitOverview, patient)

	questionIds := getQuestionIdsFromPatientVisitOverview(patientVisitOverview)
	patientAnswersForQuestions, err := p.DataApi.GetAnswersForQuestionsInPatientVisit(api.PATIENT_ROLE, questionIds, patientVisit.PatientId, patientVisit.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient answers for questions : "+err.Error())
		return
	}
	p.populatePatientVisitOverviewWithPatientAnswers(patientAnswersForQuestions, patientVisitOverview, patient)
	p.removeQuestionsWithNoAnswersBasedOnFlag(patientVisitOverview, patient)
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, DoctorPatientVisitReviewResponse{patientVisitOverview})
}

func (p *DoctorPatientVisitReviewHandler) populatePatientVisitOverviewWithPatientAnswers(patientAnswers map[int64][]*common.AnswerIntake,
	patientVisitOverview *info_intake.PatientVisitOverview,
	patient *common.Patient) {
	// collect all question ids for which to get patient answers
	for _, section := range patientVisitOverview.Sections {
		for _, subSection := range section.SubSections {
			for _, question := range subSection.Questions {
				if question.QuestionId != 0 {
					if patientAnswers[question.QuestionId] != nil {
						question.PatientAnswers = patientAnswers[question.QuestionId]
						GetSignedUrlsForAnswersInQuestion(&question.Question, p.PatientPhotoStorageService)
					}
				}
			}
		}
	}
	return
}

func (p *DoctorPatientVisitReviewHandler) filterOutGenderSpecificQuestionsAndSubSectionsFromOverview(patientVisitOverview *info_intake.PatientVisitOverview, patient *common.Patient) {
	for _, section := range patientVisitOverview.Sections {
		filteredSubSections := make([]*info_intake.PatientVisitOverviewSubSection, 0)
		for _, subSection := range section.SubSections {
			if !(subSection.GenderFilter == "" || subSection.GenderFilter == patient.Gender) {
				continue
			}
			filteredQuestions := make([]*info_intake.PatientVisitOverviewQuestion, 0)
			for _, question := range subSection.Questions {
				if question.GenderFilter == "" || question.GenderFilter == patient.Gender {
					filteredQuestions = append(filteredQuestions, question)
				}
			}
			subSection.Questions = filteredQuestions
			filteredSubSections = append(filteredSubSections, subSection)
		}
		section.SubSections = filteredSubSections
	}
}

func (p *DoctorPatientVisitReviewHandler) removeQuestionsWithNoAnswersBasedOnFlag(patientVisitOverview *info_intake.PatientVisitOverview, patient *common.Patient) {
	for _, section := range patientVisitOverview.Sections {
		for _, subSection := range section.SubSections {
			filteredQuestions := make([]*info_intake.PatientVisitOverviewQuestion, 0)
			for _, question := range subSection.Questions {
				if question.RemoveQuestionIfNoAnswer == true {
					if question.PatientAnswers != nil && len(question.PatientAnswers) > 0 {
						filteredQuestions = append(filteredQuestions, question)
					}
				} else {
					filteredQuestions = append(filteredQuestions, question)
				}
			}
			subSection.Questions = filteredQuestions
		}
	}

}

func fillInPatientVisitInfoIntoOverview(patientVisit *common.PatientVisit, patientVisitOverview *info_intake.PatientVisitOverview) {
	patientVisitOverview.PatientVisitTime = patientVisit.SubmittedDate
	patientVisitOverview.PatientVisitId = patientVisit.PatientVisitId
	patientVisitOverview.HealthConditionId = patientVisit.HealthConditionId
}

func getQuestionIdsFromPatientVisitOverview(patientVisitOverview *info_intake.PatientVisitOverview) (questionIds []int64) {
	// collect all question ids for which to get patient answers
	questionIds = make([]int64, 0)
	for _, section := range patientVisitOverview.Sections {
		for _, subSection := range section.SubSections {
			for _, question := range subSection.Questions {
				if question.QuestionId != 0 {
					questionIds = append(questionIds, question.QuestionId)
				}
			}
		}
	}
	return
}
