package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/info_intake"
	thriftapi "carefront/thrift/api"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/schema"
	"github.com/subosito/twilio"
)

const (
	HEALTH_CONDITION_ACNE_ID = 1
)

const doctorNewVisitNotification = "SPRUCE: You have a new patient visit waiting."

type PatientVisitHandler struct {
	DataApi                    api.DataAPI
	AuthApi                    thriftapi.Auth
	LayoutStorageService       api.CloudStorageAPI
	PatientPhotoStorageService api.CloudStorageAPI
	twilioCli                  *twilio.Client
	twilioFromNumber           string
}

type PatientVisitRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id,required"`
}

type PatientVisitResponse struct {
	PatientVisitId int64                         `json:"patient_visit_id,string"`
	Status         string                        `json:"status,omitempty"`
	StatusMessage  string                        `json:"status_message,omitempty"`
	ClientLayout   *info_intake.InfoIntakeLayout `json:"health_condition,omitempty"`
}

type PatientVisitSubmittedResponse struct {
	PatientVisitId int64  `json:"patient_visit_id,string"`
	Status         string `json:"status,omitempty"`
}

func NewPatientVisitHandler(dataApi api.DataAPI, authApi thriftapi.Auth, layoutStorageService api.CloudStorageAPI, patientPhotoStorageService api.CloudStorageAPI, twilioCli *twilio.Client, twilioFromNumber string) *PatientVisitHandler {
	return &PatientVisitHandler{dataApi, authApi, layoutStorageService, patientPhotoStorageService, twilioCli, twilioFromNumber}
}

func (s *PatientVisitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case HTTP_GET:
		s.returnLastCreatedPatientVisit(w, r)
	case HTTP_POST:
		s.createNewPatientVisitHandler(w, r)
	case HTTP_PUT:
		s.submitPatientVisit(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *PatientVisitHandler) submitPatientVisit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	requestData := new(PatientVisitRequestData)
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	patientId, err := s.DataApi.GetPatientIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from accountId retrieved from auth token: "+err.Error())
		return
	}

	patientIdFromPatientVisitId, err := s.DataApi.GetPatientIdFromPatientVisitId(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from patientVisitId: "+err.Error())
		return
	}

	if patientId != patientIdFromPatientVisitId {
		WriteDeveloperError(w, http.StatusBadRequest, "PatientId from auth token and patient id from patient visit don't match")
		return
	}

	patientVisit, err := s.DataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient visit from id: "+err.Error())
		return
	}

	// do not support the submitting of a case that has already been submitted or is in another state
	if patientVisit.Status != api.CASE_STATUS_OPEN && patientVisit.Status != api.CASE_STATUS_PHOTOS_REJECTED {
		WriteDeveloperError(w, http.StatusBadRequest, "Cannot submit a case that is not in the open state. Current status of case = "+patientVisit.Status)
		return
	}

	err = s.DataApi.SubmitPatientVisitWithId(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to submit patient visit to doctor for review and diagnosis: "+err.Error())
		return
	}

	patientVisit, err = s.DataApi.GetPatientVisitFromId(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusOK, "Unable to get the patient visit object based on id: "+err.Error())
	}

	careTeam, err := s.DataApi.GetCareTeamForPatient(patientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get care team for patient: "+err.Error())
		return
	}

	var doctorId int64
	for _, assignment := range careTeam.Assignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE {
			doctorId = assignment.ProviderId
			break
		}
	}

	err = s.DataApi.AssignPatientVisitToDoctor(doctorId, requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to assign patient visit to doctor")
		return
	}

	if s.twilioCli != nil {
		if doc, err := s.DataApi.GetDoctorFromId(doctorId); err != nil {
			log.Printf("Failed to get doctor for ID %d: %s", doctorId, err.Error())
		} else {
			if doc.CellPhone != "" {
				_, _, err = s.twilioCli.Messages.SendSMS(s.twilioFromNumber, doc.CellPhone, doctorNewVisitNotification)
				if err != nil {
					log.Println("Error sending SMS: " + err.Error())
				}
			}
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientVisitSubmittedResponse{PatientVisitId: patientVisit.PatientVisitId.Int64(), Status: patientVisit.Status})
}

func (s *PatientVisitHandler) returnLastCreatedPatientVisit(w http.ResponseWriter, r *http.Request) {

	patientId, err := s.DataApi.GetPatientIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from the accountId retreived from the auth token: "+err.Error())
		return
	}

	// get the last created patient visit for this patient
	patientVisitId, err := s.DataApi.GetLastCreatedPatientVisitIdForPatient(patientId)
	if err != nil {
		if err == api.NoRowsError {
			WriteDeveloperErrorWithCode(w, DEVELOPER_ERROR_NO_VISIT_EXISTS, http.StatusBadRequest, "No patient visit exists for this patient")
			return
		}

		WriteDeveloperError(w, http.StatusInternalServerError, `unable to retrieve the current active patient 
			visit for the health condition from the patient id: `+err.Error())
		return
	}

	patientVisit, err := s.DataApi.GetPatientVisitFromId(patientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient visit from id: "+err.Error())
		return
	}

	careTeam, err := s.DataApi.GetCareTeamForPatient(patientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get care team for patient")
		return
	}

	primaryDoctorId := getPrimaryDoctorIdFromCareTeam(careTeam)
	if primaryDoctorId == 0 {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to identify the primary doctor for the patient")
		return
	}
	doctor, err := s.DataApi.GetDoctorFromId(primaryDoctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the doctor from id: "+err.Error())
		return
	}

	// if there is an active patient visit record, then ensure to lookup the layout to send to the patient
	// based on what layout was shown to the patient at the time of opening of the patient visit, NOT the current
	// based on what is the current active layout because that may have potentially changed and we want to ensure
	// to not confuse the patient by changing the question structure under their feet for this particular patient visit
	// in other words, want to show them what they have already seen in terms of a flow.
	healthCondition, _, err := s.getClientLayoutForPatientVisit(patientVisitId, api.EN_LANGUAGE_ID)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get client layout for existing patient visit: "+err.Error())
		return
	}

	err = s.populateGlobalSectionsWithPatientAnswers(healthCondition, patientId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// get answers that the patient has previously entered for this particular patient visit
	// and feed the answers into the layout
	sectionIdsForHealthCondition, err := s.DataApi.GetSectionIdsForHealthCondition(HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get section ids for health condition: "+err.Error())
		return
	}
	questionIdsInAllSections := make([]int64, 0)
	for _, sectionId := range sectionIdsForHealthCondition {
		questionIds := getQuestionIdsInSectionInHealthConditionLayout(healthCondition, sectionId)
		questionIdsInAllSections = append(questionIdsInAllSections, questionIds...)
	}
	patientAnswersForVisit, err := s.DataApi.GetAnswersForQuestionsInPatientVisit(api.PATIENT_ROLE, questionIdsInAllSections, patientId, patientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient answers for patient visit: "+err.Error())
		return
	}
	s.populateHealthConditionWithPatientAnswers(healthCondition, patientAnswersForVisit)
	s.fillInFormattedFieldsForQuestions(healthCondition, doctor)

	message, err := s.DataApi.GetMessageForPatientVisitStatus(patientVisit.PatientVisitId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get message for patient visit: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientVisitResponse{PatientVisitId: patientVisitId, ClientLayout: healthCondition, Status: patientVisit.Status, StatusMessage: message})
}

func (s *PatientVisitHandler) createNewPatientVisitHandler(w http.ResponseWriter, r *http.Request) {
	patient, err := s.DataApi.GetPatientFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patientId from the accountId retreived from the auth token: "+err.Error())
		return
	}

	// get the last created patient visit for this patient
	patientVisitId, err := s.DataApi.GetLastCreatedPatientVisitIdForPatient(patient.PatientId.Int64())
	if err != nil && err != api.NoRowsError {
		WriteDeveloperError(w, http.StatusInternalServerError, `unable to retrieve the current active patient 
			visit for the health condition from the patient id: `+err.Error())
		return
	}

	if patientVisitId != 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "We are only supporting 1 patient visit per patient for now, so intentionally failing this call.")
		return
	}

	// if there isn't one, then pick the current active condition layout to send to the client for the patient to enter information
	healthCondition, layoutVersionId, err := s.getCurrentActiveClientLayoutForHealthCondition(HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get current active client digestable layout: "+err.Error())
		return
	}

	patientVisitId, err = s.DataApi.CreateNewPatientVisit(patient.PatientId.Int64(), HEALTH_CONDITION_ACNE_ID, layoutVersionId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create new patient visit id: "+err.Error())
		return
	}

	doctor, err := GetPrimaryDoctorInfoBasedOnPatient(s.DataApi, patient, "")
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor info based on patient: "+err.Error())
		return
	}

	err = s.populateGlobalSectionsWithPatientAnswers(healthCondition, patient.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.fillInFormattedFieldsForQuestions(healthCondition, doctor)
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientVisitResponse{PatientVisitId: patientVisitId, ClientLayout: healthCondition})
}

func (s *PatientVisitHandler) fillInFormattedFieldsForQuestions(healthCondition *info_intake.InfoIntakeLayout, doctor *common.Doctor) {
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

func (s *PatientVisitHandler) populateGlobalSectionsWithPatientAnswers(healthCondition *info_intake.InfoIntakeLayout, patientId int64) error {
	// identify sections that are global
	globalSectionIds, err := s.DataApi.GetGlobalSectionIds()
	if err != nil {
		return errors.New("Unable to get global sections ids: " + err.Error())
	}

	globalQuestionIds := make([]int64, 0)
	for _, sectionId := range globalSectionIds {
		questionIds := getQuestionIdsInSectionInHealthConditionLayout(healthCondition, sectionId)
		globalQuestionIds = append(globalQuestionIds, questionIds...)
	}

	// get the answers that the patient has previously entered for all sections that are considered global
	globalSectionPatientAnswers, err := s.DataApi.GetPatientAnswersForQuestionsInGlobalSections(globalQuestionIds, patientId)
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

func (s *PatientVisitHandler) populateHealthConditionWithPatientAnswers(healthCondition *info_intake.InfoIntakeLayout, patientAnswers map[int64][]*common.AnswerIntake) {
	for _, section := range healthCondition.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				// go through each question to see if there exists a patient answer for it
				if patientAnswers[question.QuestionId] != nil {
					question.PatientAnswers = patientAnswers[question.QuestionId]
					GetSignedUrlsForAnswersInQuestion(question, s.PatientPhotoStorageService)
				}
			}
		}
	}
}

func (s *PatientVisitHandler) getCurrentActiveClientLayoutForHealthCondition(healthConditionId, languageId int64) (healthCondition *info_intake.InfoIntakeLayout, layoutVersionId int64, err error) {
	var e error
	bucket, key, region, layoutVersionId, e := s.DataApi.GetStorageInfoOfCurrentActivePatientLayout(languageId, healthConditionId)
	if e != nil {
		err = e
		return
	}

	healthCondition, err = s.getHealthConditionObjectAtLocation(bucket, key, region)
	return
}

func (s *PatientVisitHandler) getClientLayoutForPatientVisit(patientVisitId, languageId int64) (healthCondition *info_intake.InfoIntakeLayout, layoutVersionId int64, err error) {
	layoutVersionId, err = s.DataApi.GetLayoutVersionIdForPatientVisit(patientVisitId)
	if err != nil {
		return
	}

	var e error
	bucket, key, region, e := s.DataApi.GetStorageInfoForClientLayout(layoutVersionId, languageId)
	if e != nil {
		err = e
		return
	}

	healthCondition, err = s.getHealthConditionObjectAtLocation(bucket, key, region)
	return
}

func (s *PatientVisitHandler) getHealthConditionObjectAtLocation(bucket, key, region string) (*info_intake.InfoIntakeLayout, error) {
	data, _, err := s.LayoutStorageService.GetObjectAtLocation(bucket, key, region)
	if err != nil {
		return nil, err
	}
	healthCondition := &info_intake.InfoIntakeLayout{}
	if err := json.Unmarshal(data, healthCondition); err != nil {
		return nil, err
	}
	return healthCondition, nil
}
