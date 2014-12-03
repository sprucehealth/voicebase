package patient

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/sku"
)

func GetPatientVisitLayout(dataApi api.DataAPI, dispatcher *dispatch.Dispatcher,
	store storage.Store, expirationDuration time.Duration,
	patientVisit *common.PatientVisit,
	r *http.Request) (*info_intake.InfoIntakeLayout, error) {

	if err := checkLayoutVersionForFollowup(dataApi, dispatcher, patientVisit, r); err != nil {
		return nil, err
	}

	// if there is an active patient visit record, then ensure to lookup the layout to send to the patient
	// based on what layout was shown to the patient at the time of opening of the patient visit, NOT the current
	// based on what is the current active layout because that may have potentially changed and we want to ensure
	// to not confuse the patient by changing the question structure under their feet for this particular patient visit
	// in other words, want to show them what they have already seen in terms of a flow.
	patientVisitLayout, err := apiservice.GetPatientLayoutForPatientVisit(patientVisit, api.EN_LANGUAGE_ID, dataApi)
	if err != nil {
		return nil, err
	}

	err = populateSectionsWithPatientAnswers(dataApi, store, expirationDuration, patientVisit.PatientId.Int64(), patientVisit.PatientVisitId.Int64(), patientVisitLayout)
	if err != nil {
		return nil, err
	}
	return patientVisitLayout, nil
}

func createPatientVisit(patient *common.Patient, dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, store storage.Store,
	expirationDuration time.Duration, r *http.Request, context *apiservice.VisitLayoutContext) (*PatientVisitResponse, error) {

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
		clientLayout, layoutVersionId, err = apiservice.GetCurrentActiveClientLayoutForHealthCondition(dataAPI,
			api.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID, sku.AcneVisit,
			sHeaders.AppVersion, sHeaders.Platform, nil)
		if err != nil {
			return nil, err
		}

		patientVisit = &common.PatientVisit{
			PatientId:         patient.PatientId,
			HealthConditionId: encoding.NewObjectId(api.HEALTH_CONDITION_ACNE_ID),
			Status:            common.PVStatusOpen,
			LayoutVersionId:   encoding.NewObjectId(layoutVersionId),
			SKU:               sku.AcneVisit,
		}

		_, err = dataAPI.CreatePatientVisit(patientVisit)
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
		clientLayout, err = GetPatientVisitLayout(dataAPI, dispatcher, store, expirationDuration, patientVisit, r)
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

func populateSectionsWithPatientAnswers(dataApi api.DataAPI, store storage.Store, expirationDuration time.Duration, patientId, patientVisitId int64, patientVisitLayout *info_intake.InfoIntakeLayout) error {
	// get answers that the patient has previously entered for this particular patient visit
	// and feed the answers into the layout
	questionIdsInAllSections := apiservice.GetNonPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout)
	photoQuestionIds := apiservice.GetPhotoQuestionIdsInPatientVisitLayout(patientVisitLayout)

	patientAnswersForVisit, err := dataApi.AnswersForQuestions(questionIdsInAllSections, &api.PatientIntake{
		PatientID:      patientId,
		PatientVisitID: patientVisitId})
	if err != nil {
		return err
	}

	photoSectionsByQuestion, err := dataApi.PatientPhotoSectionsForQuestionIDs(photoQuestionIds, patientId, patientVisitId)
	if err != nil {
		return err
	}

	for questionId, answers := range photoSectionsByQuestion {
		patientAnswersForVisit[questionId] = answers
	}

	err = populateIntakeLayoutWithPatientAnswers(dataApi, store, expirationDuration, patientVisitLayout, patientAnswersForVisit)
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

func populateIntakeLayoutWithPatientAnswers(dataApi api.DataAPI, store storage.Store, expirationDuration time.Duration, intake *info_intake.InfoIntakeLayout, patientAnswers map[int64][]common.Answer) error {
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
								media, err := dataApi.GetMedia(photoIntakeSlot.PhotoID)
								if err != nil {
									return err
								}

								if media.ClaimerID != photoIntakeSection.ID {
									return fmt.Errorf("ClaimerId does not match Photo Intake Section Id")
								}

								photoIntakeSlot.PhotoURL, err = store.GetSignedURL(media.URL, time.Now().Add(expirationDuration))
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
