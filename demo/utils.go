package demo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/messages"
	"github.com/sprucehealth/backend/patient_visit"
)

func startPatientIntakeSubmission(answersToQuestions []*apiservice.AnswerToQuestionItem, patientVisitId int64, patientAuthToken string, signal chan int, r *http.Request) {

	go func() {

		answerIntakeRequestBody := &apiservice.AnswerIntakeRequestBody{
			PatientVisitId: patientVisitId,
			Questions:      answersToQuestions,
		}

		jsonData, _ := json.Marshal(answerIntakeRequestBody)
		answerQuestionsRequest, err := http.NewRequest("POST", answerQuestionsUrl, bytes.NewReader(jsonData))
		answerQuestionsRequest.Header.Set("Content-Type", "application/json")
		answerQuestionsRequest.Header.Set("Authorization", "token "+patientAuthToken)
		answerQuestionsRequest.Host = r.Host

		resp, err := http.DefaultClient.Do(answerQuestionsRequest)
		if err != nil {
			golog.Errorf("Error while submitting patient intake: %+v", err)
			signal <- failure
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			golog.Errorf("Expected 200 got %d", resp.StatusCode)
			signal <- failure
			return
		}
		signal <- success
	}()
}

func (c *Handler) startSendingMessageToDoctor(token, message string, caseID int64, signal chan int, r *http.Request) {
	go func() {
		requestData := &messages.PostMessageRequest{
			Message: message,
			CaseID:  caseID,
		}
		jsonData, _ := json.Marshal(requestData)
		newConversationRequest, err := http.NewRequest("POST", messagesUrl, bytes.NewReader(jsonData))
		newConversationRequest.Header.Set("Content-Type", "application/json")
		newConversationRequest.Header.Set("Authorization", "token "+token)
		newConversationRequest.Host = r.Host

		resp, err := http.DefaultClient.Do(newConversationRequest)
		if err != nil {
			golog.Errorf("Error while starting new conversation for patient: %+v", err)
			signal <- failure
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			golog.Errorf("Expected 200 but got %d", resp.StatusCode)
			signal <- failure
			return
		}
		signal <- success
	}()
}

func (c *Handler) startPhotoSubmissionForPatient(questionId, patientVisitId int64, photoSections []*common.PhotoIntakeSection, patientAuthToken string, signal chan int, r *http.Request) {

	go func() {

		patient, err := c.dataApi.GetPatientFromPatientVisitId(patientVisitId)
		if err != nil {
			golog.Errorf("Unable to get patient id from patient visit id: %s", err)
			signal <- failure
			return
		}

		for _, photoSection := range photoSections {
			for _, photo := range photoSection.Photos {
				// get the key of the photo under the assumption that the caller of this method populated
				// the photo key into the photo url
				photoKey := photo.PhotoUrl

				// get the url of the image so as to add the photo to the photos table
				url := fmt.Sprintf("s3://%s/%s/%s", c.awsRegion, fmt.Sprintf(demoPhotosBucketFormat, c.environment), photoKey)

				// instead of uploading the image via the handler, short-circuiting the photo upload
				// since we are using a small pool of images. This not only saves space but also makes the
				// creation of a demo visit a lot quicker
				if photoId, err := c.dataApi.AddPhoto(patient.PersonId, url, "image/jpeg"); err != nil {
					golog.Errorf("Unable to add photo to photo table: %s ", err)
					signal <- failure
					return
				} else {
					photo.PhotoId = photoId
				}
			}
		}

		// prepare the request to submit the photo sections
		requestData := patient_visit.PhotoAnswerIntakeRequestData{
			PatientVisitId: patientVisitId,
			PhotoQuestions: []*patient_visit.PhotoAnswerIntakeQuestionItem{
				&patient_visit.PhotoAnswerIntakeQuestionItem{
					QuestionId:    questionId,
					PhotoSections: photoSections,
				},
			},
		}

		jsonData, err := json.Marshal(&requestData)
		if err != nil {
			golog.Errorf("Unable to marshal json for photo intake: %s", err)
			signal <- failure
			return
		}

		photoIntakeRequest, err := http.NewRequest("POST", photoIntakeUrl, bytes.NewReader(jsonData))
		photoIntakeRequest.Header.Set("Content-Type", "application/json")
		photoIntakeRequest.Header.Set("Authorization", "token "+patientAuthToken)
		photoIntakeRequest.Host = r.Host
		resp, err := http.DefaultClient.Do(photoIntakeRequest)
		if err != nil || resp.StatusCode != http.StatusOK {
			golog.Errorf("Error while trying submit photo for intake: %+v", err)
			signal <- failure
			return
		}
		resp.Body.Close()
		signal <- success
	}()
}

func loginAsDoctor(email string, password, host string) (string, *common.Doctor, error) {
	params := url.Values{
		"email":    []string{email},
		"password": []string{password},
	}
	loginRequest, err := http.NewRequest("POST", dAuthUrl, strings.NewReader(params.Encode()))
	if err != nil {
		return "", nil, err
	}
	loginRequest.Host = host
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(loginRequest)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("Expected 200 response intsead got %d", res.StatusCode)
	}

	responseData := &apiservice.DoctorAuthenticationResponse{}
	err = json.NewDecoder(res.Body).Decode(responseData)
	if err != nil {
		return "", nil, err
	}

	return responseData.Token, responseData.Doctor, nil
}

func reviewPatientVisit(patientVisitId int64, authHeader, host string) error {
	visitReviewRequest, err := http.NewRequest("GET", dVisitReviewUrl+"?patient_visit_id="+strconv.FormatInt(patientVisitId, 10), nil)
	if err != nil {
		return err
	}
	visitReviewRequest.Host = host
	visitReviewRequest.Header.Set("Authorization", authHeader)
	res, err := http.DefaultClient.Do(visitReviewRequest)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200 response instead got %d", res.StatusCode)
	}

	return nil
}

func pickTreatmentPlan(patientVisitId int64, authHeader, host string) (*doctor_treatment_plan.DoctorTreatmentPlanResponse, error) {
	jsonData, err := json.Marshal(&doctor_treatment_plan.PickTreatmentPlanRequestData{
		TPParent: &common.TreatmentPlanParent{
			ParentId:   encoding.NewObjectId(patientVisitId),
			ParentType: common.TPParentTypePatientVisit,
		},
	})
	if err != nil {
		return nil, err
	}

	pickATPRequest, err := http.NewRequest("POST", dTPUrl, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	pickATPRequest.Host = host
	pickATPRequest.Header.Set("Content-Type", "application/json")
	pickATPRequest.Header.Set("Authorization", authHeader)
	res, err := http.DefaultClient.Do(pickATPRequest)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Expected 200 but got %d instead", res.StatusCode)
	}

	tpResponse := &doctor_treatment_plan.DoctorTreatmentPlanResponse{}
	err = json.NewDecoder(res.Body).Decode(tpResponse)
	if err != nil {
		return nil, err
	}

	return tpResponse, nil
}

func addRegimenToTreatmentPlan(regimenPlan *common.RegimenPlan, authHeader, host string) (*common.RegimenPlan, error) {
	jsonData, err := json.Marshal(regimenPlan)
	if err != nil {
		return nil, err
	}
	addRegimenPlanRequest, err := http.NewRequest("POST", regimenUrl, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	addRegimenPlanRequest.Host = host
	addRegimenPlanRequest.Header.Set("Content-Type", "application/json")
	addRegimenPlanRequest.Header.Set("Authorization", authHeader)
	res, err := http.DefaultClient.Do(addRegimenPlanRequest)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Expected 200 instead got %d", res.StatusCode)
	}

	updatedRegimenPlan := &common.RegimenPlan{}
	err = json.NewDecoder(res.Body).Decode(&updatedRegimenPlan)

	if err != nil {
		return nil, err
	}

	return updatedRegimenPlan, nil
}

func addTreatmentsToTreatmentPlan(treatments []*common.Treatment, treatmentPlanId int64, authHeader, host string) error {
	jsonData, err := json.Marshal(doctor_treatment_plan.AddTreatmentsRequestBody{
		Treatments:      treatments,
		TreatmentPlanId: encoding.NewObjectId(treatmentPlanId),
	})
	if err != nil {
		return err
	}

	addTreatmentsRequest, err := http.NewRequest("POST", addTreatmentsUrl, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	addTreatmentsRequest.Host = host
	addTreatmentsRequest.Header.Set("Authorization", authHeader)
	addTreatmentsRequest.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(addTreatmentsRequest)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200 instead got %d", res.StatusCode)
	}
	return nil
}

func submitTreatmentPlan(treatmentPlanId int64, message, authHeader, host string) error {
	jsonData, err := json.Marshal(&doctor_treatment_plan.TreatmentPlanRequestData{
		TreatmentPlanId: encoding.NewObjectId(treatmentPlanId),
		Message:         message,
	})

	submitTPREquest, err := http.NewRequest("PUT", dTPUrl, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	submitTPREquest.Header.Set("Authorization", authHeader)
	submitTPREquest.Header.Set("Content-Type", "application/json")
	submitTPREquest.Host = host
	res, err := http.DefaultClient.Do(submitTPREquest)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200 but got %d", res.StatusCode)
	}
	return nil
}
