package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/info_intake"
	"encoding/json"
	"github.com/gorilla/schema"
	"net/http"
)

type DoctorPatientVisitReviewHandler struct {
	DataApi                    api.DataAPI
	LayoutStorageService       api.CloudStorageAPI
	PatientPhotoStorageService api.CloudStorageAPI
}

type DoctorPatientVisitReviewRequestBody struct {
	PatientVisitId int64 `schema:"patient_visit_id,required"`
}

type DoctorPatientVisitReviewResponse struct {
	DoctorLayout *info_intake.PatientVisitOverview `json:"patient_visit_overview,omitempty"`
}

// TODO: This API is temporarily nonauthenticated, as we try and figure out
// how doctor authentication works
func (p *DoctorPatientVisitReviewHandler) NonAuthenticated() bool {
	return true
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

	patientVisit, err := p.DataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient visit information from database based on provided patient visit id : "+err.Error())
		return
	}

	patient, err := p.DataApi.GetPatientFromId(patientVisit.PatientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient from the patient id: "+err.Error())
		return
	}

	bucket, key, region, _, err := p.DataApi.GetStorageInfoOfCurrentActiveDoctorLayout(patientVisit.HealthConditionId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the active layout version for the doctor's view of the patient visit: "+err.Error())
		return
	}

	data, err := p.LayoutStorageService.GetObjectAtLocation(bucket, key, region)
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

	questionIds := getQuestionIdsFromPatientVisitOverview(patientVisitOverview)
	patientAnswersForQuestions, err := p.DataApi.GetPatientAnswersForQuestionsInPatientVisit(questionIds, patientVisit.PatientId, requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient answers for questions : "+err.Error())
		return
	}
	p.populatePatientVisitOverviewWithPatientAnswers(patientAnswersForQuestions, patientVisitOverview, patient)
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, DoctorPatientVisitReviewResponse{patientVisitOverview})
}

func (p *DoctorPatientVisitReviewHandler) populatePatientVisitOverviewWithPatientAnswers(patientAnswers map[int64][]*common.PatientAnswer,
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
				} else {
					switch question.QuestionTag {
					case "q_dob":
						patientAnswer := &common.PatientAnswer{}
						patientAnswer.AnswerText = patient.Dob.String()
						question.PatientAnswers = []*common.PatientAnswer{patientAnswer}
					case "q_gender":
						patientAnswer := &common.PatientAnswer{}
						patientAnswer.AnswerText = patient.Gender
						question.PatientAnswers = []*common.PatientAnswer{patientAnswer}
					case "q_location":
						patientAnswer := &common.PatientAnswer{}
						patientAnswer.AnswerText = patient.ZipCode
						question.PatientAnswers = []*common.PatientAnswer{patientAnswer}
					}
				}
			}
		}
	}
	return
}

func fillInPatientVisitInfoIntoOverview(patientVisit *common.PatientVisit, patientVisitOverview *info_intake.PatientVisitOverview) {
	patientVisitOverview.PatientVisitTime = patientVisit.ClosedDate
	patientVisitOverview.PatientId = patientVisit.PatientId
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
