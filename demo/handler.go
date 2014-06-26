package demo

import (
	"bytes"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/golog"
	patientApiService "github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_visit"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/sprucehealth/backend/third_party/github.com/gorilla/schema"
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
	RouteToGlobalQueue          bool  `schema:"route_global"`
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
			c.createNewDemoPatient(patient, doctorId, true, requestData.RouteToGlobalQueue, message, topLevelSignal, r)
			numRemainingConversationsToStart--
		} else {
			c.createNewDemoPatient(patient, doctorId, false, requestData.RouteToGlobalQueue, "", topLevelSignal, r)
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

func (c *Handler) createNewDemoPatient(patient *common.Patient, doctorId int64, toMessageDoctor bool, routeToGlobalQueue bool, message string, topLevelSignal chan int, r *http.Request) {
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

		// only assign patient a doctor if wanting to route visit to doctor's local queue
		if !routeToGlobalQueue {
			urlValues.Set("doctor_id", fmt.Sprintf("%d", doctorId))
		}

		signupPatientRequest, err := http.NewRequest("POST", signupPatientUrl, bytes.NewBufferString(urlValues.Encode()))
		signupPatientRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		signupPatientRequest.Host = r.Host

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
		resp.Body.Close()
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
		createPatientVisitRequest.Host = r.Host
		resp, err = http.DefaultClient.Do(createPatientVisitRequest)
		if err != nil || resp.StatusCode != http.StatusOK {
			golog.Errorf("Unable to create new patient visit: %+v", err)
			topLevelSignal <- failure
			return
		}

		patientVisitResponse := &patient_visit.PatientVisitResponse{}
		err = json.NewDecoder(resp.Body).Decode(&patientVisitResponse)
		resp.Body.Close()
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
			questionIds[questionTags[questionInfoItem.QuestionTag]] = questionInfoItem.QuestionId
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

		// get the photo slots for each of the photo questions
		questionIdToPhotoSlots := make(map[questionTag][]*info_intake.PhotoSlot)
		questionIdToPhotoSlots[qFacePhotoSection], err = c.dataApi.GetPhotoSlots(questionIds[qFacePhotoSection], api.EN_LANGUAGE_ID)
		if err != nil {
			golog.Errorf("Unable to get photo slots for q_face_photo_section: %s", err)
			topLevelSignal <- failure
			return
		}
		questionIdToPhotoSlots[qChestPhotoSection], err = c.dataApi.GetPhotoSlots(questionIds[qChestPhotoSection], api.EN_LANGUAGE_ID)
		if err != nil {
			golog.Errorf("Unable to get photo slots for q_chest_photo_section: %s", err)
			topLevelSignal <- failure
			return
		}
		questionIdToPhotoSlots[qOtherLocationPhotoSection], err = c.dataApi.GetPhotoSlots(questionIds[qOtherLocationPhotoSection], api.EN_LANGUAGE_ID)
		if err != nil {
			golog.Errorf("Unable to get photo slots for q_other_location_photo_section: %s", err)
			topLevelSignal <- failure
			return
		}

		numRequestsWaitingFor := 4
		if toMessageDoctor {
			numRequestsWaitingFor = 5
		}

		// use a buffered channel so that the goroutines don't block
		// until the receiver reads off the channel
		signal := make(chan int, numRequestsWaitingFor)

		startPatientIntakeSubmission(answersToQuestions, patientVisitResponse.PatientVisitId, signupResponse.Token, signal, r)

		c.startPhotoSubmissionForPatient(questionIds[qFacePhotoSection], patientVisitResponse.PatientVisitId, []*common.PhotoIntakeSection{
			&common.PhotoIntakeSection{
				QuestionId: questionIds[qFacePhotoSection],
				Name:       "Face",
				Photos: []*common.PhotoIntakeSlot{
					&common.PhotoIntakeSlot{
						PhotoUrl: frontPhoto,
						SlotId:   questionIdToPhotoSlots[qFacePhotoSection][0].Id,
						Name:     "Front",
					},
					&common.PhotoIntakeSlot{
						PhotoUrl: profileLeftPhoto,
						SlotId:   questionIdToPhotoSlots[qFacePhotoSection][1].Id,
						Name:     "Profile Left",
					},
					&common.PhotoIntakeSlot{
						PhotoUrl: profileRightPhoto,
						SlotId:   questionIdToPhotoSlots[qFacePhotoSection][2].Id,
						Name:     "Profile Right",
					},
				},
			},
		}, signupResponse.Token, signal, r)

		c.startPhotoSubmissionForPatient(questionIds[qChestPhotoSection], patientVisitResponse.PatientVisitId, []*common.PhotoIntakeSection{
			&common.PhotoIntakeSection{
				QuestionId: questionIds[qChestPhotoSection],
				Name:       "Chest",
				Photos: []*common.PhotoIntakeSlot{
					&common.PhotoIntakeSlot{
						PhotoUrl: chestPhoto,
						SlotId:   questionIdToPhotoSlots[qChestPhotoSection][0].Id,
						Name:     "Chest",
					},
				},
			},
		}, signupResponse.Token, signal, r)

		c.startPhotoSubmissionForPatient(questionIds[qOtherLocationPhotoSection], patientVisitResponse.PatientVisitId, []*common.PhotoIntakeSection{
			&common.PhotoIntakeSection{
				QuestionId: questionIds[qOtherLocationPhotoSection],
				Name:       "Arm",
				Photos: []*common.PhotoIntakeSlot{
					&common.PhotoIntakeSlot{
						PhotoUrl: frontPhoto,
						SlotId:   questionIdToPhotoSlots[qOtherLocationPhotoSection][0].Id,
						Name:     "Right Arm",
					},
					&common.PhotoIntakeSlot{
						PhotoUrl: profileLeftPhoto,
						SlotId:   questionIdToPhotoSlots[qOtherLocationPhotoSection][0].Id,
						Name:     "Left Arm",
					},
				},
			},
		}, signupResponse.Token, signal, r)

		if toMessageDoctor {
			caseID, err := c.dataApi.GetPatientCaseIdFromPatientVisitId(patientVisitResponse.PatientVisitId)
			if err != nil {
				golog.Errorf("Failed to get case ID for visit")
			} else {
				c.startSendingMessageToDoctor(signupResponse.Token, message, caseID, signal, r)
			}
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
		submitPatientVisitRequest.Host = r.Host
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
