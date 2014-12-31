package demo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/golog"
	patientAPIService "github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_visit"
)

type Worker struct {
	dataAPI                api.DataAPI
	lockAPI                api.LockAPI
	apiDomain              string
	awsRegion              string
	timePeriodInSeconds    int
	questionIDs            map[questionTag]int64
	questionIdToPhotoSlots map[questionTag][]*info_intake.PhotoSlot
	answerIds              map[potentialAnswerTag]int64
	stopChan               chan bool
}

const (
	defaultTimePeriodSeconds = 20
	totalPendingSets         = 5
)

func NewWorker(
	dataAPI api.DataAPI,
	lockAPI api.LockAPI,
	apiDomain, awsRegion string) *Worker {

	return &Worker{
		dataAPI:             dataAPI,
		lockAPI:             lockAPI,
		awsRegion:           awsRegion,
		apiDomain:           apiDomain,
		timePeriodInSeconds: defaultTimePeriodSeconds,
		stopChan:            make(chan bool),
	}
}

func (w *Worker) Start() {
	go func() {

		if err := w.CacheQAInformation(); err != nil {
			golog.Errorf("Unable to cache q/a information on start: %s", err)
			return
		}

		defer w.lockAPI.Release()
		for {
			if !w.lockAPI.Wait() {
				return
			}

			select {
			case <-w.stopChan:
				return
			default:
			}

			if err := w.Do(); err != nil {
				golog.Errorf(err.Error())
			}

			select {
			case <-w.stopChan:
				return
			case <-time.After(time.Duration(w.timePeriodInSeconds) * time.Second):
			}

		}
	}()
}

func (w *Worker) Stop() {
	close(w.stopChan)
}

func (w *Worker) Do() error {

	// determine the number of training cases to create based on the number that exist
	pendingSets, err := w.dataAPI.TrainingCaseSetCount(common.TCSStatusPending)
	if err != nil {
		return err
	}

	numSetsToCreate := totalPendingSets - pendingSets

	// nothing to do if no sets to create
	if numSetsToCreate <= 0 {
		return nil
	}

	for i := 0; i < numSetsToCreate; i++ {
		if err := w.createTrainingCaseSet(); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) createTrainingCaseSet() error {
	trainingCaseSetID, err := w.dataAPI.CreateTrainingCaseSet(common.TCSStatusCreating)
	if err != nil {
		return err
	}

	// iterate through each of the cases and queue up a training case for each
	for _, trainingCase := range trainingCases {

		// ********** CREATE RANDOM PATIENT **********
		// Note that once this random patient is created, we will use the patientId and the accountId
		// to update the patient information. The reason to go through this flow instead of directly
		// adding the patient to the database is to avoid the work of assigning a care team to the patient
		// and setting a patient up with an account
		randomNumber, err := common.GenerateRandomNumber(99999, 5)
		if err != nil {
			return err
		}
		urlValues := url.Values{}
		urlValues.Set("first_name", trainingCase.PatientToCreate.FirstName)
		urlValues.Set("last_name", trainingCase.PatientToCreate.LastName)
		urlValues.Set("dob", trainingCase.PatientToCreate.DOB.String())
		urlValues.Set("gender", trainingCase.PatientToCreate.Gender)
		urlValues.Set("zip_code", trainingCase.PatientToCreate.ZipCode)
		urlValues.Set("phone", trainingCase.PatientToCreate.PhoneNumbers[0].Phone.String())
		urlValues.Set("password", "12345")
		urlValues.Set("email", fmt.Sprintf("%s-%s@example.com", trainingCase.Name, randomNumber))
		urlValues.Set("training", "true")
		signupPatientRequest, err := http.NewRequest("POST", LocalServerURL+signupPatientUrl, bytes.NewBufferString(urlValues.Encode()))
		signupPatientRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		signupPatientRequest.Host = w.apiDomain
		resp, err := http.DefaultClient.Do(signupPatientRequest)
		if err != nil {
			return err
		} else if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("create visit: expected 200 but got %d", resp.StatusCode)
		}

		signupResponse := &patientAPIService.PatientSignedupResponse{}
		err = json.NewDecoder(resp.Body).Decode(&signupResponse)
		resp.Body.Close()
		if err != nil {
			return err
		}

		// ******* UPDATE PATIENT INFORMATION TO ADD ADDRESS AND PHARMACY *******
		trainingCase.PatientToCreate.PatientID = signupResponse.Patient.PatientID
		trainingCase.PatientToCreate.AccountID = signupResponse.Patient.AccountID
		trainingCase.PatientToCreate.Email = signupResponse.Patient.Email
		err = w.dataAPI.UpdatePatientInformation(trainingCase.PatientToCreate, false)
		if err != nil {
			return err
		}

		err = w.dataAPI.UpdatePatientPharmacy(trainingCase.PatientToCreate.PatientID.Int64(), trainingCase.PatientToCreate.Pharmacy)
		if err != nil {
			return err
		}

		// ********** CREATE PATIENT VISIT **********
		createPatientVisitRequest, err := http.NewRequest("POST", LocalServerURL+patientVisitUrl, nil)
		createPatientVisitRequest.Header.Set("Authorization", "token "+signupResponse.Token)
		createPatientVisitRequest.Host = w.apiDomain
		createPatientVisitRequest.Header.Set("S-Version", "Patient;Dev;1.0")
		createPatientVisitRequest.Header.Set("S-OS", "iOS")
		resp, err = http.DefaultClient.Do(createPatientVisitRequest)
		if err != nil {
			return err
		} else if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("create visit: expected 200 but got %d", resp.StatusCode)
		}

		patientVisitResponse := &patientAPIService.PatientVisitResponse{}
		err = json.NewDecoder(resp.Body).Decode(&patientVisitResponse)
		resp.Body.Close()
		if err != nil {
			return err
		}

		// ********** SIMULATE PATIENT INTAKE **********
		answersToQuestions := populatePatientIntake(w.questionIDs, w.answerIds, trainingCase.IntakeToSubmit)

		if err := w.submitAnswersForVisit(answersToQuestions,
			patientVisitResponse.PatientVisitID,
			signupResponse.Token); err != nil {
			return err
		}

		for _, photoIntake := range trainingCase.PhotoSectionsToSubmit {
			pSection := &common.PhotoIntakeSection{
				QuestionID: w.questionIDs[photoIntake.QuestionTag],
				Name:       photoIntake.SectionName,
				Photos:     make([]*common.PhotoIntakeSlot, len(photoIntake.PhotoSlots)),
			}

			for j, slot := range photoIntake.PhotoSlots {
				pSection.Photos[j] = &common.PhotoIntakeSlot{
					PhotoURL: slot.PhotoURL,
					Name:     slot.Name,
					SlotID:   w.questionIdToPhotoSlots[photoIntake.QuestionTag][0].ID,
				}
			}

			if err := w.submitPhotosForVisit(w.questionIDs[photoIntake.QuestionTag],
				patientVisitResponse.PatientVisitID,
				[]*common.PhotoIntakeSection{pSection},
				signupResponse.Token); err != nil {
				return err
			}
		}

		if trainingCase.VisitMessage != "" {
			if err := w.submitMessageForVisit(signupResponse.Token,
				trainingCase.VisitMessage,
				patientVisitResponse.PatientVisitID); err != nil {
				return err
			}
		}

		// ********** SUBMIT CASE TO DOCTOR **********
		submitPatientVisitRequest, err := http.NewRequest("PUT", LocalServerURL+patientVisitUrl, bytes.NewBufferString(fmt.Sprintf("patient_visit_id=%d", patientVisitResponse.PatientVisitID)))
		if err != nil {
			return err
		}

		submitPatientVisitRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		submitPatientVisitRequest.Header.Set("Authorization", "token "+signupResponse.Token)
		submitPatientVisitRequest.Host = w.apiDomain
		resp, err = http.DefaultClient.Do(submitPatientVisitRequest)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("submit visit: expected 200 but got %d", resp.StatusCode)
		}

		// Now that it has been submitted go ahead and add it to the training case set
		if err := w.dataAPI.QueueTrainingCase(&common.TrainingCase{
			TrainingCaseSetID: trainingCaseSetID,
			PatientVisitID:    patientVisitResponse.PatientVisitID,
			TemplateName:      trainingCase.Name,
		}); err != nil {
			return err
		}
	}

	// make the case set active after all cases in the set have been added
	if err := w.dataAPI.UpdateTrainingCaseSetStatus(trainingCaseSetID,
		common.TCSStatusPending); err != nil {
		return err
	}

	return nil
}

func (w *Worker) CacheQAInformation() error {
	// cache question and answer information on start

	w.questionIDs = make(map[questionTag]int64)
	questionTagsForLookup := make([]string, 0)
	for questionTagString, _ := range questionTags {
		questionTagsForLookup = append(questionTagsForLookup, questionTagString)
	}
	questionInfos, err := w.dataAPI.GetQuestionInfoForTags(questionTagsForLookup, api.EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	for _, questionInfoItem := range questionInfos {
		w.questionIDs[questionTags[questionInfoItem.QuestionTag]] = questionInfoItem.QuestionID
	}

	w.answerIds = make(map[potentialAnswerTag]int64)
	answerTagsForLookup := make([]string, 0)
	for answerTagString, _ := range answerTags {
		answerTagsForLookup = append(answerTagsForLookup, answerTagString)
	}
	answerInfos, err := w.dataAPI.GetAnswerInfoForTags(answerTagsForLookup, api.EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	for _, answerInfoItem := range answerInfos {
		w.answerIds[answerTags[answerInfoItem.AnswerTag]] = answerInfoItem.AnswerID
	}

	w.questionIdToPhotoSlots = make(map[questionTag][]*info_intake.PhotoSlot)
	w.questionIdToPhotoSlots[qFacePhotoSection], err = w.dataAPI.GetPhotoSlots(w.questionIDs[qFacePhotoSection], api.EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	w.questionIdToPhotoSlots[qChestPhotoSection], err = w.dataAPI.GetPhotoSlots(w.questionIDs[qChestPhotoSection], api.EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	w.questionIdToPhotoSlots[qBackPhotoSection], err = w.dataAPI.GetPhotoSlots(w.questionIDs[qBackPhotoSection], api.EN_LANGUAGE_ID)
	if err != nil {
		return err
	}
	w.questionIdToPhotoSlots[qOtherLocationPhotoSection], err = w.dataAPI.GetPhotoSlots(w.questionIDs[qOtherLocationPhotoSection], api.EN_LANGUAGE_ID)
	if err != nil {
		return err
	}

	return nil
}

func populatePatientIntake(questionIDs map[questionTag]int64, answerIds map[potentialAnswerTag]int64, answerTemplates map[questionTag][]*answerTemplate) []*apiservice.QuestionAnswerItem {
	answerIntake := make([]*apiservice.QuestionAnswerItem, 0, len(answerTemplates))
	for questionTag, templates := range answerTemplates {
		aItem := &apiservice.QuestionAnswerItem{
			QuestionID: questionIDs[questionTag],
		}
		aItem.AnswerIntakes = make([]*apiservice.AnswerItem, len(templates))
		for i, template := range templates {

			aItem.AnswerIntakes[i] = &apiservice.AnswerItem{}

			if template.AnswerText != "" {
				aItem.AnswerIntakes[i].AnswerText = template.AnswerText
			}

			if answerIds[template.AnswerTag] != 0 {
				aItem.AnswerIntakes[i].PotentialAnswerID = answerIds[template.AnswerTag]
			}

			if len(template.SubquestionAnswers) > 0 {
				aItem.AnswerIntakes[i].SubQuestions = make([]*apiservice.QuestionAnswerItem, len(template.SubquestionAnswers))
				var subAnswerItems []*apiservice.QuestionAnswerItem
				for _, subAnswerTemplates := range template.SubquestionAnswers {
					subAnswerItems = append(subAnswerItems, populatePatientIntake(questionIDs, answerIds, subAnswerTemplates)...)
				}
				for j, subAnswerItem := range subAnswerItems {
					aItem.AnswerIntakes[i].SubQuestions[j] = &apiservice.QuestionAnswerItem{
						QuestionID:    subAnswerItem.QuestionID,
						AnswerIntakes: subAnswerItem.AnswerIntakes,
					}
				}
			}
		}

		answerIntake = append(answerIntake, aItem)
	}

	return answerIntake
}

func (w *Worker) submitAnswersForVisit(answersToQuestions []*apiservice.QuestionAnswerItem, patientVisitID int64, patientAuthToken string) error {

	intakeData := &apiservice.IntakeData{
		PatientVisitID: patientVisitID,
		Questions:      answersToQuestions,
	}

	jsonData, err := json.Marshal(intakeData)
	if err != nil {
		return err
	}
	answerQuestionsRequest, err := http.NewRequest("POST", LocalServerURL+answerQuestionsUrl, bytes.NewReader(jsonData))
	answerQuestionsRequest.Header.Set("Content-Type", "application/json")
	answerQuestionsRequest.Header.Set("Authorization", "token "+patientAuthToken)
	answerQuestionsRequest.Host = w.apiDomain

	resp, err := http.DefaultClient.Do(answerQuestionsRequest)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200 got %d", resp.StatusCode)
	}

	return nil
}

func (w *Worker) submitPhotosForVisit(questionID, patientVisitID int64, photoSections []*common.PhotoIntakeSection, patientAuthToken string) error {
	patient, err := w.dataAPI.GetPatientFromPatientVisitID(patientVisitID)
	if err != nil {
		return err
	}

	for _, photoSection := range photoSections {
		for _, photo := range photoSection.Photos {

			// get the url of the image so as to add the photo to the photos table
			url := fmt.Sprintf("s3://%s/%s/%s", w.awsRegion, fmt.Sprintf(demoPhotosBucketFormat, environment.GetCurrent()), photo.PhotoURL)

			// instead of uploading the image via the handler, short-circuiting the photo upload
			// since we are using a small pool of images. This not only saves space but also makes the
			// creation of a demo visit a lot quicker
			if photoId, err := w.dataAPI.AddMedia(patient.PersonID, url, "image/jpeg"); err != nil {
				return err
			} else {
				photo.PhotoID = photoId
			}
		}
	}

	// prepare the request to submit the photo sections
	requestData := patient_visit.PhotoAnswerIntakeRequestData{
		PatientVisitID: patientVisitID,
		PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
			&patient_visit.PhotoAnswerIntakeQuestionItem{
				QuestionID:    questionID,
				PhotoSections: photoSections,
			},
		},
	}

	jsonData, err := json.Marshal(&requestData)
	if err != nil {
		return err
	}

	photoIntakeRequest, err := http.NewRequest("POST", LocalServerURL+photoIntakeUrl, bytes.NewReader(jsonData))
	photoIntakeRequest.Header.Set("Content-Type", "application/json")
	photoIntakeRequest.Header.Set("Authorization", "token "+patientAuthToken)
	photoIntakeRequest.Host = w.apiDomain
	resp, err := http.DefaultClient.Do(photoIntakeRequest)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("photo submission: expected 200 but got %d", resp.StatusCode)
	}
	return nil
}

func (w *Worker) submitMessageForVisit(token, message string, visitID int64) error {
	requestData := map[string]interface{}{
		"visit_id": strconv.FormatInt(visitID, 10),
		"message":  message,
	}
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", LocalServerURL+visitMessageUrl, bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+token)
	req.Host = w.apiDomain

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200 but got %d", resp.StatusCode)
	}
	return nil
}
