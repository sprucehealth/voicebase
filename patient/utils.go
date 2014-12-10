package patient

import (
	"errors"
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

func IntakeLayoutForVisit(
	dataAPI api.DataAPI,
	store storage.Store,
	expirationDuration time.Duration,
	visit *common.PatientVisit) (*info_intake.InfoIntakeLayout, error) {

	// if there is an active patient visit record, then ensure to lookup the layout to send to the patient
	// based on what layout was shown to the patient at the time of opening of the patient visit, NOT the current
	// based on what is the current active layout because that may have potentially changed and we want to ensure
	// to not confuse the patient by changing the question structure under their feet for this particular patient visit
	// in other words, want to show them what they have already seen in terms of a flow.
	visitLayout, err := apiservice.GetPatientLayoutForPatientVisit(visit, api.EN_LANGUAGE_ID, dataAPI)
	if err != nil {
		return nil, err
	}

	err = populateLayoutWithAnswers(
		visitLayout,
		dataAPI,
		store,
		expirationDuration,
		visit)

	return visitLayout, err
}

func populateLayoutWithAnswers(
	visitLayout *info_intake.InfoIntakeLayout,
	dataAPI api.DataAPI,
	store storage.Store,
	expirationDuration time.Duration,
	patientVisit *common.PatientVisit,
) error {

	patientID := patientVisit.PatientID.Int64()
	visitID := patientVisit.PatientVisitID.Int64()

	photoQuestionIDs := visitLayout.PhotoQuestionIDs()
	photosForVisit, err := dataAPI.PatientPhotoSectionsForQuestionIDs(photoQuestionIDs, patientID, visitID)
	if err != nil {
		return err
	}

	// create photoURLs for each answer
	expirationTime := time.Now().Add(expirationDuration)
	for _, photoSections := range photosForVisit {
		for _, photoSection := range photoSections {
			ps := photoSection.(*common.PhotoIntakeSection)
			for _, intakeSlot := range ps.Photos {
				media, err := dataAPI.GetMedia(intakeSlot.PhotoID)
				if err != nil {
					return err
				}

				if ok, err := dataAPI.MediaHasClaim(intakeSlot.PhotoID, common.ClaimerTypePhotoIntakeSection, ps.ID); err != nil {
					return err
				} else if !ok {
					return errors.New("ClaimerID does not match PhotoIntakeSectionID")
				}

				intakeSlot.PhotoURL, err = store.GetSignedURL(media.URL, expirationTime)
				if err != nil {
					return err
				}
			}
		}

	}

	nonPhotoQuestionIDs := visitLayout.NonPhotoQuestionIDs()
	answersForVisit, err := dataAPI.AnswersForQuestions(nonPhotoQuestionIDs, &api.PatientIntake{
		PatientID:      patientID,
		PatientVisitID: visitID,
	})
	if err != nil {
		return err
	}

	// merge answers into one map
	for questionID, answers := range photosForVisit {
		answersForVisit[questionID] = answers
	}

	// keep track of any question that is to be prefilled
	// and doesn't have an answer for this visit yet
	prefillQuestionsWithNoAnswers := make(map[int64]*info_intake.Question)
	var prefillQuestionIDs []int64
	// populate layout with the answers for each question
	for _, section := range visitLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				question.Answers = answersForVisit[question.QuestionID]
				if question.ToPrefill && len(question.Answers) == 0 {
					prefillQuestionsWithNoAnswers[question.QuestionID] = question
					prefillQuestionIDs = append(prefillQuestionIDs, question.QuestionID)
				}
			}
		}
	}

	// if visit is still open, prefill any questions currently unanswered
	// with answers by the patient from a previous visit
	if patientVisit.Status == common.PVStatusOpen {
		previousAnswers, err := dataAPI.PreviousPatientAnswersForQuestions(
			prefillQuestionIDs, patientID, patientVisit.CreationDate)
		if err != nil {
			return err
		}

		// populate the questions with previous answers by the patient
		for questionID, answers := range previousAnswers {
			prefillQuestionsWithNoAnswers[questionID].PrefilledWithPreviousAnswers = true
			prefillQuestionsWithNoAnswers[questionID].Answers = answers
		}
	}

	return nil
}

func createPatientVisit(patient *common.Patient, dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, store storage.Store,
	expirationDuration time.Duration, r *http.Request, context *apiservice.VisitLayoutContext) (*PatientVisitResponse, error) {

	var clientLayout *info_intake.InfoIntakeLayout

	// get the last created patient visit for this patient
	patientVisit, err := dataAPI.GetLastCreatedPatientVisit(patient.PatientID.Int64())
	if err != nil && err != api.NoRowsError {
		return nil, err
	} else if err == nil && patientVisit.Status != common.PVStatusOpen {
		return nil, apiservice.NewValidationError("We are only supporting 1 patient visit per patient for now, so intentionally failing this call.", r)
	}

	if patientVisit == nil {
		// start a new visit
		var layoutVersionID int64
		sHeaders := apiservice.ExtractSpruceHeaders(r)
		clientLayout, layoutVersionID, err = apiservice.GetCurrentActiveClientLayoutForHealthCondition(dataAPI,
			api.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID, sku.AcneVisit,
			sHeaders.AppVersion, sHeaders.Platform, nil)
		if err != nil {
			return nil, err
		}

		patientVisit = &common.PatientVisit{
			PatientID:         patient.PatientID,
			HealthConditionID: encoding.NewObjectID(api.HEALTH_CONDITION_ACNE_ID),
			Status:            common.PVStatusOpen,
			LayoutVersionID:   encoding.NewObjectID(layoutVersionID),
			SKU:               sku.AcneVisit,
		}

		_, err = dataAPI.CreatePatientVisit(patientVisit)
		if err != nil {
			return nil, err
		}

		dispatcher.Publish(&VisitStartedEvent{
			PatientID:     patient.PatientID.Int64(),
			VisitID:       patientVisit.PatientVisitID.Int64(),
			PatientCaseID: patientVisit.PatientCaseID.Int64(),
		})
	} else {
		// return current visit
		clientLayout, err = IntakeLayoutForVisit(dataAPI, store, expirationDuration, patientVisit)
		if err != nil {
			return nil, err
		}
	}

	return &PatientVisitResponse{
		PatientVisitID: patientVisit.PatientVisitID.Int64(),
		Status:         patientVisit.Status,
		ClientLayout:   clientLayout,
	}, nil
}
