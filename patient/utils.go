package patient

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/media"
)

func IntakeLayoutForVisit(
	dataAPI api.DataAPI,
	apiDomain string,
	mediaStore *media.Store,
	expirationDuration time.Duration,
	visit *common.PatientVisit) (*VisitIntakeInfo, error) {

	errs := make(chan error, 2)
	var visitLayout *info_intake.InfoIntakeLayout
	var doctorID int64
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var err error

		// if there is an active patient visit record, then ensure to lookup the layout to send to the patient
		// based on what layout was shown to the patient at the time of opening of the patient visit, NOT the current
		// based on what is the current active layout because that may have potentially changed and we want to ensure
		// to not confuse the patient by changing the question structure under their feet for this particular patient visit
		// in other words, want to show them what they have already seen in terms of a flow.
		visitLayout, err = apiservice.GetPatientLayoutForPatientVisit(visit, api.EN_LANGUAGE_ID, dataAPI, apiDomain)
		if err != nil {
			errs <- err
		}

		if err := populateLayoutWithAnswers(
			visitLayout,
			dataAPI,
			mediaStore,
			expirationDuration,
			visit); err != nil {
			errs <- err
		}
	}()

	go func() {
		defer wg.Done()

		doctorMember, err := dataAPI.GetActiveCareTeamMemberForCase(api.DOCTOR_ROLE, visit.PatientCaseID.Int64())
		if err != nil && !api.IsErrNotFound(err) {
			errs <- err
		}

		if doctorMember != nil {
			doctorID = doctorMember.ProviderID
		}
	}()

	wg.Wait()

	select {
	case err := <-errs:
		return nil, err
	default:
	}

	return &VisitIntakeInfo{
		PatientVisitID: visit.PatientVisitID.Int64(),
		CanAbandon:     !visit.IsFollowup,
		Status:         visit.Status,
		ClientLayout:   visitLayout,
		DoctorID:       doctorID,
	}, nil
}

func populateLayoutWithAnswers(
	visitLayout *info_intake.InfoIntakeLayout,
	dataAPI api.DataAPI,
	mediaStore *media.Store,
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
	for _, photoSections := range photosForVisit {
		for _, photoSection := range photoSections {
			ps := photoSection.(*common.PhotoIntakeSection)
			for _, intakeSlot := range ps.Photos {
				if ok, err := dataAPI.MediaHasClaim(intakeSlot.PhotoID, common.ClaimerTypePhotoIntakeSection, ps.ID); err != nil {
					return err
				} else if !ok {
					return errors.New("ClaimerID does not match PhotoIntakeSectionID")
				}

				intakeSlot.PhotoURL, err = mediaStore.SignedURL(intakeSlot.PhotoID, expirationDuration)
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

func createPatientVisit(
	patient *common.Patient,
	doctorID int64,
	pathwayTag string,
	dataAPI api.DataAPI,
	apiDomain string,
	dispatcher *dispatch.Dispatcher,
	mediaStore *media.Store,
	expirationDuration time.Duration,
	r *http.Request,
	context *apiservice.VisitLayoutContext,
) (*PatientVisitResponse, error) {

	var patientVisit *common.PatientVisit

	patientCases, err := dataAPI.CasesForPathway(patient.PatientID.Int64(), pathwayTag, []string{common.PCStatusOpen.String(), common.PCStatusActive.String()})
	if err != nil {
		return nil, err
	} else if err == nil {
		switch l := len(patientCases); {
		case l == 0:
		case l == 1:
			// if there exists open visits against an active case for this pathwayTag, return
			// the last created patient visit. Technically, the patient should not have more than a single open
			// patient visit against a case.
			patientVisits, err := dataAPI.GetVisitsForCase(patientCases[0].ID.Int64(), common.OpenPatientVisitStates())
			if err != nil {
				return nil, err
			} else if len(patientVisits) > 0 {
				sort.Reverse(common.ByPatientVisitCreationDate(patientVisits))
				patientVisit = patientVisits[0]
			}
		default:
			return nil, fmt.Errorf("Only a single active case per pathway can exist for now. Pathway %s has %d active cases.", pathwayTag, len(patientCases))
		}
	}

	visitCreated := false
	if patientVisit == nil {
		pathway, err := dataAPI.PathwayForTag(pathwayTag, api.PONone)
		if err != nil {
			return nil, err
		}

		sku, err := dataAPI.SKUForPathway(pathwayTag, common.SCVisit)
		if err != nil {
			return nil, err
		}

		// start a new visit
		sHeaders := apiservice.ExtractSpruceHeaders(r)
		layoutVersionID, err := dataAPI.IntakeLayoutVersionIDForAppVersion(
			sHeaders.AppVersion,
			sHeaders.Platform,
			pathway.ID,
			api.EN_LANGUAGE_ID,
			sku.Type)
		if err != nil {
			return nil, err
		}

		patientVisit = &common.PatientVisit{
			PatientID:       patient.PatientID,
			PathwayTag:      pathway.Tag,
			Status:          common.PVStatusOpen,
			LayoutVersionID: encoding.NewObjectID(layoutVersionID),
			SKUType:         sku.Type,
		}

		var dID *int64
		if doctorID != 0 {
			dID = &doctorID
		}
		_, err = dataAPI.CreatePatientVisit(patientVisit, dID)
		if err != nil {
			return nil, err
		}

		// assign the doctor to the case if the doctor is specified
		if doctorID > 0 {
			if err := dataAPI.AddDoctorToPatientCase(doctorID, patientVisit.PatientCaseID.Int64()); err != nil {
				return nil, err
			}
		}
		visitCreated = true
	}

	intakeInfo, err := IntakeLayoutForVisit(dataAPI, apiDomain, mediaStore, expirationDuration, patientVisit)
	if err != nil {
		return nil, err
	}

	if visitCreated {
		dispatcher.Publish(&VisitStartedEvent{
			PatientID:     patient.PatientID.Int64(),
			VisitID:       patientVisit.PatientVisitID.Int64(),
			PatientCaseID: patientVisit.PatientCaseID.Int64(),
		})
	}

	return &PatientVisitResponse{
		VisitIntakeInfo: intakeInfo,
	}, nil
}
