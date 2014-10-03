package patient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/storage"
)

func createPatientVisit(patient *common.Patient, dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, store storage.Store,
	expirationDuration time.Duration, r *http.Request) (*PatientVisitResponse, error) {

	var clientLayout *info_intake.InfoIntakeLayout

	// get the last created patient visit for this patient
	patientVisit, err := dataAPI.GetLastCreatedPatientVisit(patient.PatientId.Int64())
	if err != nil && err != api.NoRowsError {
		return nil, err
	} else if err == nil && patientVisit.Status != common.PVStatusOpen {
		return nil, apiservice.NewValidationError("We are only supporting 1 patient visit per patient for now, so intentionally failing this call.", r)
	}

	if patientVisit == nil {
		// start a new visit
		var layoutVersionId int64
		sHeaders := apiservice.ExtractSpruceHeaders(r)
		clientLayout, layoutVersionId, err = getCurrentActiveClientLayoutForHealthCondition(dataAPI,
			apiservice.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID,
			sHeaders.AppVersion, sHeaders.Platform)
		if err != nil {
			return nil, err
		}

		patientVisit, err = dataAPI.CreateNewPatientVisit(patient.PatientId.Int64(), apiservice.HEALTH_CONDITION_ACNE_ID, layoutVersionId)
		if err != nil {
			return nil, err
		}

		err = populateGlobalSectionsWithPatientAnswers(dataAPI, store, expirationDuration, clientLayout, patient.PatientId.Int64(), r)
		if err != nil {
			return nil, err
		}

		dispatcher.Publish(&VisitStartedEvent{
			PatientId:     patient.PatientId.Int64(),
			VisitId:       patientVisit.PatientVisitId.Int64(),
			PatientCaseId: patientVisit.PatientCaseId.Int64(),
		})
	} else {
		// return current visit
		clientLayout, err = GetPatientVisitLayout(dataAPI, store, expirationDuration, patientVisit, r)
		if err != nil {
			return nil, err
		}
	}

	return &PatientVisitResponse{
		PatientVisitId: patientVisit.PatientVisitId.Int64(),
		Status:         patientVisit.Status,
		ClientLayout:   clientLayout,
	}, nil
}

func populateGlobalSectionsWithPatientAnswers(dataApi api.DataAPI, store storage.Store, expirationDuration time.Duration, healthCondition *info_intake.InfoIntakeLayout, patientId int64, r *http.Request) error {
	// identify sections that are global
	globalSectionIds, err := dataApi.GetGlobalSectionIds()
	if err != nil {
		return errors.New("Unable to get global sections ids: " + err.Error())
	}

	globalQuestionIds := make([]int64, 0)
	for _, sectionId := range globalSectionIds {
		questionIds := getQuestionIdsInSectionInIntakeLayout(healthCondition, sectionId)
		globalQuestionIds = append(globalQuestionIds, questionIds...)
	}

	// get the answers that the patient has previously entered for all sections that are considered global
	globalSectionPatientAnswers, err := dataApi.GetPatientAnswersForQuestionsInGlobalSections(globalQuestionIds, patientId)
	if err != nil {
		return errors.New("Unable to get patient answers for global sections: " + err.Error())
	}

	err = populateIntakeLayoutWithPatientAnswers(dataApi, store, expirationDuration, healthCondition, globalSectionPatientAnswers, r)
	if err != nil {
		return err
	}
	return nil
}

func populateSectionsWithPatientAnswers(dataApi api.DataAPI, store storage.Store, expirationDuration time.Duration, patientId, patientVisitId int64, patientVisitLayout *info_intake.InfoIntakeLayout, r *http.Request) error {
	// get answers that the patient has previously entered for this particular patient visit
	// and feed the answers into the layout
	questionIdsInAllSections := apiservice.GetNonPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout)
	photoQuestionIds := apiservice.GetPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout)

	patientAnswersForVisit, err := dataApi.GetPatientAnswersForQuestions(questionIdsInAllSections, patientId, patientVisitId)
	if err != nil {
		return err
	}

	photoSectionsByQuestion, err := dataApi.GetPatientCreatedPhotoSectionsForQuestionIds(photoQuestionIds, patientId, patientVisitId)
	if err != nil {
		return err
	}

	for questionId, answers := range photoSectionsByQuestion {
		patientAnswersForVisit[questionId] = answers
	}

	err = populateIntakeLayoutWithPatientAnswers(dataApi, store, expirationDuration, patientVisitLayout, patientAnswersForVisit, r)
	if err != nil {
		return err
	}
	return nil
}

func getQuestionIdsInSectionInIntakeLayout(healthCondition *info_intake.InfoIntakeLayout, sectionId int64) (questionIds []int64) {
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

func populateIntakeLayoutWithPatientAnswers(dataApi api.DataAPI, store storage.Store, expirationDuration time.Duration, intake *info_intake.InfoIntakeLayout, patientAnswers map[int64][]common.Answer, r *http.Request) error {
	for _, section := range intake.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				// go through each question to see if there exists a patient answer for it
				question.Answers = patientAnswers[question.QuestionId]
				if question.QuestionType == info_intake.QUESTION_TYPE_PHOTO_SECTION {
					if len(question.Answers) > 0 {
						// go through each slot and populate the url for the photo
						for _, answer := range question.Answers {
							photoIntakeSection := answer.(*common.PhotoIntakeSection)
							for _, photoIntakeSlot := range photoIntakeSection.Photos {
								media, err := dataApi.GetMedia(photoIntakeSlot.PhotoId)
								if err != nil {
									return err
								}

								if media.ClaimerID != photoIntakeSection.Id {
									return fmt.Errorf("ClaimerId does not match Photo Intake Section Id")
								}

								photoIntakeSlot.PhotoUrl, err = store.GetSignedURL(media.URL, time.Now().Add(expirationDuration))
								if err != nil {
									return err
								}
							}
						}
					}

				}
			}
		}
	}
	return nil
}

func getCurrentActiveClientLayoutForHealthCondition(dataApi api.DataAPI, healthConditionId, languageId int64, appVersion *common.Version, platform common.Platform) (*info_intake.InfoIntakeLayout, int64, error) {
	data, layoutVersionId, err := dataApi.IntakeLayoutForAppVersion(appVersion, platform, languageId, healthConditionId)
	if err != nil {
		return nil, 0, err
	}

	patientVisitLayout := &info_intake.InfoIntakeLayout{}
	if err := json.Unmarshal(data, patientVisitLayout); err != nil {
		return nil, 0, err
	}
	return patientVisitLayout, layoutVersionId, nil
}
