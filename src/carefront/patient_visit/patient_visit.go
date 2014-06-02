package patient_visit

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/info_intake"
	"carefront/libs/dispatch"
	thriftapi "carefront/thrift/api"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type patientVisitHandler struct {
	dataApi api.DataAPI
	authApi thriftapi.Auth
}

type patientVisitRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id,required"`
}

type PatientVisitResponse struct {
	PatientVisitId int64                         `json:"patient_visit_id,string"`
	Status         string                        `json:"status,omitempty"`
	ClientLayout   *info_intake.InfoIntakeLayout `json:"health_condition,omitempty"`
}

type PatientVisitSubmittedResponse struct {
	PatientVisitId int64  `json:"patient_visit_id,string"`
	Status         string `json:"status,omitempty"`
}

func NewPatientVisitHandler(dataApi api.DataAPI, authApi thriftapi.Auth) *patientVisitHandler {
	return &patientVisitHandler{
		dataApi: dataApi,
		authApi: authApi,
	}
}

func (s *patientVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		s.returnLastCreatedPatientVisit(w, r)
	case apiservice.HTTP_POST:
		s.createNewPatientVisitHandler(w, r)
	case apiservice.HTTP_PUT:
		s.submitPatientVisit(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *patientVisitHandler) submitPatientVisit(w http.ResponseWriter, r *http.Request) {
	requestData := &patientVisitRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientId, err := s.dataApi.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	patientIdFromPatientVisitId, err := s.dataApi.GetPatientIdFromPatientVisitId(requestData.PatientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if patientId != patientIdFromPatientVisitId {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "PatientId from auth token and patient id from patient visit don't match")
		return
	}

	patientVisit, err := s.dataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	// do not support the submitting of a case that has already been submitted or is in another state
	if patientVisit.Status != api.CASE_STATUS_OPEN && patientVisit.Status != api.CASE_STATUS_PHOTOS_REJECTED {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Cannot submit a case that is not in the open state. Current status of case = "+patientVisit.Status)
		return
	}

	err = s.dataApi.SubmitPatientVisitWithId(requestData.PatientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	patientVisit, err = s.dataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	careTeam, err := s.dataApi.GetCareTeamForPatient(patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var doctorId int64
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE {
			doctorId = assignment.ProviderId
			break
		}
	}

	dispatch.Default.Publish(&VisitSubmittedEvent{
		PatientId: patientId,
		DoctorId:  doctorId,
		VisitId:   patientVisit.PatientVisitId.Int64(),
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientVisitSubmittedResponse{PatientVisitId: patientVisit.PatientVisitId.Int64(), Status: patientVisit.Status})
}

func (s *patientVisitHandler) returnLastCreatedPatientVisit(w http.ResponseWriter, r *http.Request) {

	patientId, err := s.dataApi.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// get the last created patient visit for this patient
	patientVisitId, err := s.dataApi.GetLastCreatedPatientVisitIdForPatient(patientId)
	if err != nil {
		if err == api.NoRowsError {
			apiservice.WriteDeveloperErrorWithCode(w, apiservice.DEVELOPER_ERROR_NO_VISIT_EXISTS, http.StatusBadRequest, "No patient visit exists for this patient")
			return
		}

		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	patientVisit, err := s.dataApi.GetPatientVisitFromId(patientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	careTeam, err := s.dataApi.GetCareTeamForPatient(patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get care team for patient")
		return
	}

	primaryDoctorId := apiservice.GetPrimaryDoctorIdFromCareTeam(careTeam)
	if primaryDoctorId == 0 {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to identify the primary doctor for the patient")
		return
	}
	doctor, err := s.dataApi.GetDoctorFromId(primaryDoctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// if there is an active patient visit record, then ensure to lookup the layout to send to the patient
	// based on what layout was shown to the patient at the time of opening of the patient visit, NOT the current
	// based on what is the current active layout because that may have potentially changed and we want to ensure
	// to not confuse the patient by changing the question structure under their feet for this particular patient visit
	// in other words, want to show them what they have already seen in terms of a flow.
	patientVisitLayout, _, err := apiservice.GetPatientLayoutForPatientVisit(patientVisitId, api.EN_LANGUAGE_ID, s.dataApi)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = s.populateGlobalSectionsWithPatientAnswers(patientVisitLayout, patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// get answers that the patient has previously entered for this particular patient visit
	// and feed the answers into the layout
	questionIdsInAllSections := apiservice.GetQuestionIdsInPatientVisitLayout(patientVisitLayout)

	patientAnswersForVisit, err := s.dataApi.GetPatientAnswersForQuestionsBasedOnQuestionIds(questionIdsInAllSections, patientId, patientVisitId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient answers for patient visit: "+err.Error())
		return
	}

	s.populateHealthConditionWithPatientAnswers(patientVisitLayout, patientAnswersForVisit)
	s.fillInFormattedFieldsForQuestions(patientVisitLayout, doctor)

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientVisitResponse{PatientVisitId: patientVisitId, ClientLayout: patientVisitLayout, Status: patientVisit.Status})
}

func (s *patientVisitHandler) createNewPatientVisitHandler(w http.ResponseWriter, r *http.Request) {
	patient, err := s.dataApi.GetPatientFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from the accountId retreived from the auth token: "+err.Error())
		return
	}

	// get the last created patient visit for this patient
	patientVisitId, err := s.dataApi.GetLastCreatedPatientVisitIdForPatient(patient.PatientId.Int64())
	if err != nil && err != api.NoRowsError {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if patientVisitId != 0 {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "We are only supporting 1 patient visit per patient for now, so intentionally failing this call.")
		return
	}

	// if there isn't one, then pick the current active condition layout to send to the client for the patient to enter information
	healthCondition, layoutVersionId, err := s.getCurrentActiveClientLayoutForHealthCondition(apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	patientVisitId, err = s.dataApi.CreateNewPatientVisit(patient.PatientId.Int64(), apiservice.HEALTH_CONDITION_ACNE_ID, layoutVersionId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	doctor, err := apiservice.GetPrimaryDoctorInfoBasedOnPatient(s.dataApi, patient, "")
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = s.populateGlobalSectionsWithPatientAnswers(healthCondition, patient.PatientId.Int64())
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.fillInFormattedFieldsForQuestions(healthCondition, doctor)

	dispatch.Default.PublishAsync(&VisitStartedEvent{
		PatientId: patient.PatientId.Int64(),
		VisitId:   patientVisitId,
	})
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientVisitResponse{PatientVisitId: patientVisitId, ClientLayout: healthCondition})
}

func (s *patientVisitHandler) fillInFormattedFieldsForQuestions(healthCondition *info_intake.InfoIntakeLayout, doctor *common.Doctor) {
	for _, section := range healthCondition.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {

				if question.FormattedFieldTags != nil {

					// populate the values for each of the fields in order
					for _, fieldTag := range question.FormattedFieldTags {
						fieldTagComponents := strings.Split(fieldTag, ":")
						if fieldTagComponents[0] == info_intake.FORMATTED_TITLE_FIELD {
							switch fieldTagComponents[1] {
							case info_intake.FORMATTED_FIELD_DOCTOR_LAST_NAME:
								// build the formatted string and assign it back to the question title
								question.QuestionTitle = fmt.Sprintf(question.QuestionTitle, strings.Title(doctor.LastName))
							}
						}
					}

				}
			}
		}
	}
}

func (s *patientVisitHandler) populateGlobalSectionsWithPatientAnswers(healthCondition *info_intake.InfoIntakeLayout, patientId int64) error {
	// identify sections that are global
	globalSectionIds, err := s.dataApi.GetGlobalSectionIds()
	if err != nil {
		return errors.New("Unable to get global sections ids: " + err.Error())
	}

	globalQuestionIds := make([]int64, 0)
	for _, sectionId := range globalSectionIds {
		questionIds := getQuestionIdsInSectionInHealthConditionLayout(healthCondition, sectionId)
		globalQuestionIds = append(globalQuestionIds, questionIds...)
	}

	// get the answers that the patient has previously entered for all sections that are considered global
	globalSectionPatientAnswers, err := s.dataApi.GetPatientAnswersForQuestionsInGlobalSections(globalQuestionIds, patientId)
	if err != nil {
		return errors.New("Unable to get patient answers for global sections: " + err.Error())
	}

	s.populateHealthConditionWithPatientAnswers(healthCondition, globalSectionPatientAnswers)
	return nil
}

func getQuestionIdsInSectionInHealthConditionLayout(healthCondition *info_intake.InfoIntakeLayout, sectionId int64) (questionIds []int64) {
	questionIds = make([]int64, 0)
	for _, section := range healthCondition.Sections {
		if section.SectionId == sectionId {
			for _, screen := range section.Screens {
				for _, question := range screen.Questions {
					questionIds = append(questionIds, question.QuestionId)
				}
			}
		}
	}
	return
}

func (s *patientVisitHandler) populateHealthConditionWithPatientAnswers(healthCondition *info_intake.InfoIntakeLayout, patientAnswers map[int64][]*common.AnswerIntake) {
	for _, section := range healthCondition.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				// go through each question to see if there exists a patient answer for it
				if patientAnswers[question.QuestionId] != nil {
					question.Answers = patientAnswers[question.QuestionId]
					// apiservice.GetSignedUrlsForAnswersInQuestion(question, s.PatientPhotoStorageService)
				}
			}
		}
	}
}

func (s *patientVisitHandler) getCurrentActiveClientLayoutForHealthCondition(healthConditionId, languageId int64) (*info_intake.InfoIntakeLayout, int64, error) {
	data, layoutVersionId, err := s.dataApi.GetCurrentActivePatientLayout(languageId, healthConditionId)
	if err != nil {
		return nil, 0, err
	}

	patientVisitLayout := &info_intake.InfoIntakeLayout{}
	if err := json.Unmarshal(data, patientVisitLayout); err != nil {
		return nil, 0, err
	}
	return patientVisitLayout, layoutVersionId, nil
}
