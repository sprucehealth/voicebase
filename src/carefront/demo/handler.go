package demo

import (
	"bytes"
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/golog"
	patientApiService "carefront/patient"
	"carefront/patient_visit"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/schema"
)

type Handler struct {
	environment     string
	dataApi         api.DataAPI
	cloudStorageApi api.CloudStorageAPI
	awsRegion       string
}

func NewHandler(dataApi api.DataAPI, cloudStorageApi api.CloudStorageAPI, awsRegion, environment string) *Handler {
	return &Handler{
		environment:     environment,
		dataApi:         dataApi,
		cloudStorageApi: cloudStorageApi,
		awsRegion:       awsRegion,
	}
}

type CreateDemoPatientVisitRequestData struct {
	ToCreateSurescriptsPatients bool  `schema:"surescripts"`
	NumPatients                 int64 `schema:"num_patients"`
	NumConversations            int64 `schema:"num_conversations"`
}

func (c *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	doctorId, err := c.dataApi.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to get doctor based on the account id: "+err.Error())
		return
	}

	// ensure that are not working with a non-prod environment
	if c.environment == "prod" {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Cannot work in the production environment")
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse input parameters: "+err.Error())
		return
	}

	requestData := &CreateDemoPatientVisitRequestData{}
	if err := schema.NewDecoder().Decode(requestData, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse input parameters: "+err.Error())
		return
	}

	var patients []*common.Patient
	if requestData.ToCreateSurescriptsPatients {
		patients = prepareSurescriptsPatients()
	} else {
		if numPatients := requestData.NumPatients; numPatients > 0 {
			patients = prepareDemoPatients(requestData.NumPatients)
		} else {
			patients = prepareDemoPatients(1)
		}
	}

	numRemainingConversationsToStart := requestData.NumConversations
	topLevelSignal := make(chan int, len(patients))
	for i, patient := range patients {
		if numRemainingConversationsToStart > 0 {
			message := sampleMessages[i%3]
			c.createNewDemoPatient(patient, doctorId, true, message, topLevelSignal)
			numRemainingConversationsToStart--
		} else {
			c.createNewDemoPatient(patient, doctorId, false, "", topLevelSignal)
		}

		time.Sleep(500 * time.Millisecond)
	}

	numberPatientsWaitingFor := len(patients)
	for numberPatientsWaitingFor > 0 {
		result := <-topLevelSignal
		if result == failure {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Something went wrong while trying to create demo patient")
			return
		}
		numberPatientsWaitingFor--
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}

func (c *Handler) createNewDemoPatient(patient *common.Patient, doctorId int64, toMessageDoctor bool, message string, topLevelSignal chan int) {
	go func() {

		// ********** CREATE RANDOM PATIENT **********
		// Note that once this random patient is created, we will use the patientId and the accountId
		// to update the patient information. The reason to go through this flow instead of directly
		// adding the patient to the database is to avoid the work of assigning a care team to the patient
		// and setting a patient up with an account
		urlValues := url.Values{}
		urlValues.Set("first_name", patient.FirstName)
		urlValues.Set("last_name", patient.LastName)
		urlValues.Set("dob", patient.Dob.String())
		urlValues.Set("gender", patient.Gender)
		urlValues.Set("zip_code", patient.ZipCode)
		urlValues.Set("phone", patient.PhoneNumbers[0].Phone)
		urlValues.Set("password", "12345")
		urlValues.Set("email", fmt.Sprintf("%d%d@example.com", time.Now().UnixNano(), doctorId))
		urlValues.Set("doctor_id", fmt.Sprintf("%d", doctorId))
		signupPatientRequest, err := http.NewRequest("POST", signupPatientUrl, bytes.NewBufferString(urlValues.Encode()))
		signupPatientRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(signupPatientRequest)
		if err != nil {
			golog.Errorf("Unable to signup random patient:%+v", err)
			topLevelSignal <- failure
			return
		}

		if resp.StatusCode != http.StatusOK {
			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				golog.Errorf("Unable to signup random patient and unable to read body of response: %+v", err)
				topLevelSignal <- failure
				return
			}
			golog.Errorf("Status %d when trying to signup random patient: %+v", resp.StatusCode, string(respBody))
			topLevelSignal <- failure
			return
		}

		signupResponse := &patientApiService.PatientSignedupResponse{}
		err = json.NewDecoder(resp.Body).Decode(&signupResponse)
		defer resp.Body.Close()

		if err != nil {
			golog.Errorf("Unable to unmarshal response body into object: %+v", err)
			topLevelSignal <- failure
			return
		}

		// ********** UPDATE PATIENT DEMOGRAPHIC INFORMATION AS THOUGH A DOCTOR WERE UPDATING IT **********
		patient.PatientId = signupResponse.Patient.PatientId
		patient.AccountId = signupResponse.Patient.AccountId
		patient.Email = signupResponse.Patient.Email
		err = c.dataApi.UpdatePatientInformation(patient, false)
		if err != nil {
			golog.Errorf("Unable to update patient information:%+v", err)
			topLevelSignal <- failure
			return
		}

		err = c.dataApi.UpdatePatientPharmacy(patient.PatientId.Int64(), patient.Pharmacy)
		if err != nil {
			golog.Errorf("Unable to update patients preferred pharmacy:%+v", err)
			topLevelSignal <- failure
			return
		}

		// ********** CREATE PATIENT VISIT **********

		// create patient visit
		createPatientVisitRequest, err := http.NewRequest("POST", patientVisitUrl, nil)
		createPatientVisitRequest.Header.Set("Authorization", "token "+signupResponse.Token)
		resp, err = http.DefaultClient.Do(createPatientVisitRequest)
		if err != nil || resp.StatusCode != http.StatusOK {
			golog.Errorf("Unable to create new patient visit: %+v", err)
			topLevelSignal <- failure
			return
		}

		patientVisitResponse := &patient_visit.PatientVisitResponse{}
		err = json.NewDecoder(resp.Body).Decode(&patientVisitResponse)
		defer resp.Body.Close()
		if err != nil {
			golog.Errorf("Unable to unmarshal response into patient visit response: %+v", err.Error())
			topLevelSignal <- failure
			return
		}

		// ********** SIMULATE PATIENT INTAKE **********

		questionIds := make(map[questionTag]int64)
		questionTagsForLookup := make([]string, 0)
		for questionTagString, _ := range questionTags {
			questionTagsForLookup = append(questionTagsForLookup, questionTagString)
		}

		questionInfos, err := c.dataApi.GetQuestionInfoForTags(questionTagsForLookup, api.EN_LANGUAGE_ID)
		if err != nil {
			golog.Errorf("Unable to lookup ids based on question tags:%+v", err.Error())
			topLevelSignal <- failure
			return
		}

		for _, questionInfoItem := range questionInfos {
			questionIds[questionTags[questionInfoItem.QuestionTag]] = questionInfoItem.Id
		}

		answerIds := make(map[potentialAnswerTag]int64)
		answerTagsForLookup := make([]string, 0)
		for answerTagString, _ := range answerTags {
			answerTagsForLookup = append(answerTagsForLookup, answerTagString)
		}
		answerInfos, err := c.dataApi.GetAnswerInfoForTags(answerTagsForLookup, api.EN_LANGUAGE_ID)
		if err != nil {
			golog.Errorf("Unable to lookup answer infos based on tags:%+v", err.Error())
			topLevelSignal <- failure
			return
		}
		for _, answerInfoItem := range answerInfos {
			answerIds[answerTags[answerInfoItem.AnswerTag]] = answerInfoItem.AnswerId
		}

		answersToQuestions := populatePatientIntake(questionIds, answerIds)

		numRequestsWaitingFor := 5
		if toMessageDoctor {
			numRequestsWaitingFor = 6
		}

		// use a buffered channel so that the goroutines don't block
		// until the receiver reads off the channel
		signal := make(chan int, numRequestsWaitingFor)

		startPatientIntakeSubmission(answersToQuestions, patientVisitResponse.PatientVisitId, signupResponse.Token, signal)

		c.startPhotoSubmissionForPatient(questionIds[qFacePhotoIntake],
			answerIds[aFaceFrontPhotoIntake], patientVisitResponse.PatientVisitId, frontPhoto, signupResponse.Token, signal)

		c.startPhotoSubmissionForPatient(questionIds[qFaceRightPhotoIntake],
			answerIds[aProfileRightPhotoIntake], patientVisitResponse.PatientVisitId, profileRightPhoto, signupResponse.Token, signal)

		c.startPhotoSubmissionForPatient(questionIds[qFaceLeftPhotoIntake],
			answerIds[aProfileLeftPhotoIntake], patientVisitResponse.PatientVisitId, profileLeftPhoto, signupResponse.Token, signal)

		c.startPhotoSubmissionForPatient(questionIds[qChestPhotoIntake],
			answerIds[aChestPhotoIntake], patientVisitResponse.PatientVisitId, chestPhoto, signupResponse.Token, signal)

		if toMessageDoctor {
			c.startSendingMessageToDoctor(signupResponse.Token, message, signal)
		}

		// wait for all requests to finish
		for numRequestsWaitingFor > 0 {
			result := <-signal
			if result == failure {
				golog.Errorf("Something went wrong when tryign to submit patient visit intake. Patient visit not successfully submitted")
				topLevelSignal <- failure
				return

			}
			numRequestsWaitingFor--
		}

		// ********** SUBMIT CASE TO DOCTOR **********
		submitPatientVisitRequest, err := http.NewRequest("PUT", patientVisitUrl, bytes.NewBufferString(fmt.Sprintf("patient_visit_id=%d", patientVisitResponse.PatientVisitId)))
		submitPatientVisitRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		submitPatientVisitRequest.Header.Set("Authorization", "token "+signupResponse.Token)
		if err != nil {
			golog.Errorf("Unable to create new request to submit patient visit:%+v", err)
			topLevelSignal <- failure
			return
		}

		resp, err = http.DefaultClient.Do(submitPatientVisitRequest)
		if err != nil || resp.StatusCode != http.StatusOK {
			golog.Errorf("Unable to make successful request to submit patient visit")
			topLevelSignal <- failure
			return
		}

		topLevelSignal <- success
	}()

}
