package patient

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/media"
)

// IntakeLayoutForVisit returns the intake layout info for the provided visit.
func IntakeLayoutForVisit(
	dataAPI api.DataAPI,
	apiDomain string,
	webDomain string,
	mediaStore *media.Store,
	expirationDuration time.Duration,
	visit *common.PatientVisit,
	patient *common.Patient,
	viewerRole string,
) (*VisitIntakeInfo, error) {

	var visitLayout *info_intake.InfoIntakeLayout
	var doctorID int64
	var msg string

	p := conc.NewParallel()
	p.Go(func() error {
		// if there is an active patient visit record, then ensure to lookup the layout to send to the patient
		// based on what layout was shown to the patient at the time of opening of the patient visit, NOT the current
		// based on what is the current active layout because that may have potentially changed and we want to ensure
		// to not confuse the patient by changing the question structure under their feet for this particular patient visit
		// in other words, want to show them what they have already seen in terms of a flow.
		var err error
		visitLayout, err = apiservice.GetPatientLayoutForPatientVisit(visit, api.LanguageIDEnglish, dataAPI, apiDomain)
		if err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(populateLayoutWithAnswers(visitLayout, dataAPI, mediaStore, expirationDuration, visit))
	})
	p.Go(func() error {
		doctorMember, err := dataAPI.GetActiveCareTeamMemberForCase(api.RoleDoctor, visit.PatientCaseID.Int64())
		if err != nil && !api.IsErrNotFound(err) {
			return errors.Trace(err)
		}
		if doctorMember != nil {
			doctorID = doctorMember.ProviderID
		}
		return nil
	})
	p.Go(func() error {
		var err error
		msg, err = dataAPI.GetMessageForPatientVisit(visit.ID.Int64())
		if err != nil && !api.IsErrNotFound(err) {
			return errors.Trace(err)
		}
		return nil
	})
	if err := p.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	additionalMessage := &AdditionalMessage{
		VisitMessage: visitLayout.DeprecatedAdditionalMessage,
		Message:      msg,
	}

	var title string
	if visitLayout.Header != nil {
		title = visitLayout.Header.Title
	}

	info := &VisitIntakeInfo{
		PatientVisitID: visit.ID.Int64(),
		CanAbandon:     !visit.IsFollowup,
		Status:         visit.Status,
		IsSubmitted:    common.PatientVisitSubmitted(visit.Status),
		ClientLayout: &clientLayout{
			InfoIntakeLayout: visitLayout,
		},
		DoctorID:                doctorID,
		RequireCreditCardIfFree: false,
		SKUType:                 visitLayout.DeprecatedSKUType,
		AdditionalMessage:       additionalMessage,
		SubmissionConfirmation:  visitLayout.DeprecatedSubmissionConfirmation,
		Checkout:                visitLayout.DeprecatedCheckout,
		Title:                   title,
	}

	if patient != nil {
		info.ParentalConsentRequired = patient.IsUnder18()
		info.ParentalConsentGranted = patient.HasParentalConsent
		if viewerRole == api.RolePatient && info.ParentalConsentRequired && !info.ParentalConsentGranted {
			actionURL, err := ParentalConsentRequestSMSAction(dataAPI, webDomain, patient.ID.Int64())
			if err != nil {
				return nil, errors.Trace(err)
			}
			info.ParentalConsentInfo = &ParentalConsentInfo{
				ScreenTitle: "Parental Consent",
				FooterText:  "Your parent will have access to your visit, treatment plan and messages with your care team.",
				Body: ParentalConsentInfoBody{
					Title:        "Text your parent a link to get their consent for your visit.",
					IconURL:      app_url.IconConsentLarge,
					Message:      "Before submitting your visit, we need a parent to consent to your treatment. As part of their approval, your parent will need to provide a valid photo ID.",
					ButtonText:   "Text Link",
					ButtonAction: actionURL,
				},
			}
			info.ClientLayout.ParentalConsentInfo = info.ParentalConsentInfo
		}
	}

	return info, nil
}

func populateLayoutWithAnswers(
	visitLayout *info_intake.InfoIntakeLayout,
	dataAPI api.DataAPI,
	mediaStore *media.Store,
	expirationDuration time.Duration,
	patientVisit *common.PatientVisit,
) error {

	patientID := patientVisit.PatientID.Int64()
	visitID := patientVisit.ID.Int64()

	photoQuestionIDs := visitLayout.PhotoQuestionIDs()
	photosForVisit, err := dataAPI.PatientPhotoSectionsForQuestionIDs(photoQuestionIDs, patientID, visitID)
	if err != nil {
		return errors.Trace(err)
	}

	// create photoURLs for each answer
	for _, photoSections := range photosForVisit {
		for _, photoSection := range photoSections {
			ps := photoSection.(*common.PhotoIntakeSection)
			for _, intakeSlot := range ps.Photos {
				if ok, err := dataAPI.MediaHasClaim(intakeSlot.PhotoID, common.ClaimerTypePhotoIntakeSection, ps.ID); err != nil {
					return errors.Trace(err)
				} else if !ok {
					return errors.Trace(errors.New("ClaimerID does not match PhotoIntakeSectionID"))
				}

				intakeSlot.PhotoURL, err = mediaStore.SignedURL(intakeSlot.PhotoID, expirationDuration)
				if err != nil {
					return errors.Trace(err)
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
		return errors.Trace(err)
	}

	// merge answers into one map
	for questionID, answers := range photosForVisit {
		answersForVisit[questionID] = answers
	}

	// keep track of any question that is to be prefilled
	// and doesn't have an answer for this visit yet
	questionsToPrefill := make(map[string]*info_intake.Question)
	var prefillQuestionTags []string
	// populate layout with the answers for each question
	for _, section := range visitLayout.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				question.Answers = answersForVisit[question.QuestionID]
				if question.ToPrefill && len(question.Answers) == 0 {
					questionsToPrefill[question.QuestionTag] = question
					prefillQuestionTags = append(prefillQuestionTags, question.QuestionTag)
				}
			}
		}
	}

	// if visit is still open, prefill any questions currently unanswered
	// with answers by the patient from a previous visit
	if patientVisit.Status == common.PVStatusOpen {

		previousAnswers, err := dataAPI.PreviousPatientAnswersForQuestions(
			prefillQuestionTags, patientID, patientVisit.CreationDate)
		if err != nil {
			return errors.Trace(err)
		}

		// populate the questions with previous answers by the patient
		for questionTag, answers := range previousAnswers {

			populatedAnswers, err := populateAnswers(questionsToPrefill[questionTag], answers)
			if err != nil {
				return errors.Trace(err)
			}

			questionsToPrefill[questionTag].Answers = populatedAnswers
			questionsToPrefill[questionTag].PrefilledWithPreviousAnswers = populatedAnswers != nil
		}
	}

	return nil
}

func populateAnswers(question *info_intake.Question, answers []common.Answer) ([]common.Answer, error) {
	items := make([]common.Answer, len(answers))
	for i, answer := range answers {
		switch a := answer.(type) {
		case *common.AnswerIntake:

			ai := &common.AnswerIntake{
				AnswerIntakeID:   a.AnswerIntakeID,
				QuestionID:       a.QuestionID,
				ParentQuestionID: a.ParentQuestionID,
				ParentAnswerID:   a.ParentAnswerID,
				AnswerText:       a.AnswerText,
				PotentialAnswer:  a.PotentialAnswer,
				AnswerSummary:    a.AnswerSummary,
				Type:             a.Type,
			}

			// populate the potential answer id from the question versus the answer
			// so that we can map it to one of the potential answers of the new version of the question
			for _, pa := range question.PotentialAnswers {
				if pa.Answer == a.PotentialAnswer {
					ai.PotentialAnswerID = encoding.NewObjectID(pa.AnswerID)
				}
			}

			// Dont populate any answers if the patient's answer indicates that they picked a potential
			// answer which does not match any of the potential answers in the current set for the question.
			if a.PotentialAnswerID.Int64() != 0 && ai.PotentialAnswerID.Int64() == 0 {
				return nil, nil
			}

			items[i] = ai

		default:
			return nil, errors.Trace(fmt.Errorf("Expected answer of type common.AnswerIntake but got type %T", a))
		}
	}

	return items, nil
}

// pathwayForPatient validates the age of the patient against the pathway's restrictions and
// optionally returns an alternate pathway if the age range specifies one.
func pathwayForPatient(dataAPI api.DataAPI, pathwayTag string, patient *common.Patient) (*common.Pathway, error) {
	pathway, err := dataAPI.PathwayForTag(pathwayTag, api.POWithDetails)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Make sure that the patient is elligible for the selected pathway (e.g. age), and
	// see if their is an alternate pathway matching their age (e.g. teen acne).
	if pathway.Details != nil && len(pathway.Details.AgeRestrictions) != 0 {
		age := patient.DOB.Age()
		// The age restrictions are guaranteed to be ordered .
		var ageRes *common.PathwayAgeRestriction
		for _, ar := range pathway.Details.AgeRestrictions {
			if ar.MaxAgeOfRange == nil || age <= *ar.MaxAgeOfRange {
				// Only the last range should be nil so it's a catch all if nothing else has yet matched.
				ageRes = ar
				break
			}
		}
		// One range should have matched so this is just a sanity check.
		if ageRes == nil {
			return nil, fmt.Errorf("age ranges for pathway %s are invalid", pathway.Tag)
		}
		if !ageRes.VisitAllowed {
			// The app shouldn't have let the patient start a visit and should have shown this same alert message, but
			// checking it on the server side as well is always good practice.
			return nil, &apiservice.SpruceError{
				DeveloperError: fmt.Sprintf("Pathway %s does not allow patients with age %d to start a visit", pathway.Tag, age),
				UserError:      ageRes.Alert.Message,
				HTTPStatusCode: http.StatusBadRequest,
			}
		}
		// The age ranges can specify an alternate pathway that should be used to start a visit.
		// This unfortunate situations occurs because when someone is first shown the pathway menu we don't
		// yet have their age so can't direct them to the appropriate pathway.
		if ageRes.AlternatePathwayTag != "" {
			pathway, err = dataAPI.PathwayForTag(ageRes.AlternatePathwayTag, api.PONone)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	} else if patient.IsUnder18() {
		// For pathways without explicit age restrictions don't allow anyone under 18
		return nil, &apiservice.SpruceError{
			DeveloperError: "No explicit age ranges listed so not allowing anyone under 18.",
			UserError:      "Sorry, we do not support the chosen condition for people under 18.",
			HTTPStatusCode: http.StatusBadRequest,
		}
	}
	return pathway, nil
}

func createPatientVisit(
	patient *common.Patient,
	doctorID int64,
	pathwayTag string,
	dataAPI api.DataAPI,
	apiDomain string,
	webDomain string,
	dispatcher *dispatch.Dispatcher,
	mediaStore *media.Store,
	expirationDuration time.Duration,
	r *http.Request,
	context *apiservice.VisitLayoutContext,
) (*PatientVisitResponse, error) {
	// We have to resolve the pathway first because it's possible that for the patient's
	// age they might need to be taken to an alternate pathway.
	pathway, err := pathwayForPatient(dataAPI, pathwayTag, patient)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var patientVisit *common.PatientVisit

	// First check for cases on the pathway the patient chose from the menu and if none then
	// check for them against the possible alternate pathway based on age
	patientCases, err := dataAPI.CasesForPathway(patient.ID.Int64(), pathwayTag, []string{common.PCStatusOpen.String(), common.PCStatusActive.String()})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(patientCases) == 0 && pathway.Tag != pathwayTag {
		patientCases, err = dataAPI.CasesForPathway(patient.ID.Int64(), pathway.Tag, []string{common.PCStatusOpen.String(), common.PCStatusActive.String()})
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	if n := len(patientCases); n == 1 {
		// if there exists open visits against an active case for this pathwayTag, return
		// the last created patient visit. Technically, the patient should not have more than a single open
		// patient visit against a case.
		patientVisits, err := dataAPI.GetVisitsForCase(patientCases[0].ID.Int64(), common.OpenPatientVisitStates())
		if err != nil {
			return nil, errors.Trace(err)
		} else if len(patientVisits) > 0 {
			sort.Sort(sort.Reverse(common.ByPatientVisitCreationDate(patientVisits)))
			patientVisit = patientVisits[0]
		}
	} else if n != 0 {
		return nil, errors.Trace(fmt.Errorf("Only a single active case per pathway can exist for now. Pathway %s has %d active cases.", pathway.Tag, len(patientCases)))
	}

	visitCreated := false
	if patientVisit == nil {
		sku, err := dataAPI.SKUForPathway(pathway.Tag, common.SCVisit)
		if err != nil {
			return nil, errors.Trace(err)
		}

		// start a new visit
		sHeaders := apiservice.ExtractSpruceHeaders(r)
		layoutVersionID, err := dataAPI.IntakeLayoutVersionIDForAppVersion(
			sHeaders.AppVersion,
			sHeaders.Platform,
			pathway.ID,
			api.LanguageIDEnglish,
			sku.Type)
		if err != nil {
			return nil, errors.Trace(err)
		}

		patientVisit = &common.PatientVisit{
			PatientID:       patient.ID,
			PathwayTag:      pathway.Tag,
			Status:          common.PVStatusOpen,
			LayoutVersionID: encoding.NewObjectID(layoutVersionID),
			SKUType:         sku.Type,
		}

		_, err = dataAPI.CreatePatientVisit(patientVisit, ptr.Int64NilZero(doctorID))
		if err != nil {
			return nil, errors.Trace(err)
		}

		// assign the doctor to the case if the doctor is specified
		if doctorID > 0 {
			// FIXME: if there's an error at this point then the visit is still created but the VisitStartedEvent is
			// never published this there's no notification for the visit. As such, if this fails then it's better
			// to log the error but continue since the patient will be able to continue the visit anyway.
			if err := dataAPI.AddDoctorToPatientCase(doctorID, patientVisit.PatientCaseID.Int64()); err != nil {
				golog.Errorf("Failed to add doctor %d to patient case %d: %s", doctorID, patientVisit.PatientCaseID.Int64(), err)
			}
		}
		visitCreated = true
	}

	intakeInfo, err := IntakeLayoutForVisit(dataAPI, apiDomain, webDomain, mediaStore, expirationDuration, patientVisit, patient, api.RolePatient)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if visitCreated {
		dispatcher.Publish(&VisitStartedEvent{
			PatientID:     patient.ID.Int64(),
			VisitID:       patientVisit.ID.Int64(),
			PatientCaseID: patientVisit.PatientCaseID.Int64(),
		})
	}

	return &PatientVisitResponse{
		VisitIntakeInfo: intakeInfo,
	}, nil
}

func showFeedback(dataAPI api.DataAPI, patientID int64) bool {
	tp, err := latestActiveTreatmentPlan(dataAPI, patientID)
	if err != nil {
		golog.Errorf(err.Error())
		return false
	}
	if tp == nil || !tp.PatientViewed {
		return false
	}

	feedbackFor := "case:" + strconv.FormatInt(tp.PatientCaseID.Int64(), 10)
	recorded, err := dataAPI.PatientFeedbackRecorded(patientID, feedbackFor)
	if err != nil {
		golog.Errorf("Failed to get feedback for patient %d %s: %s", patientID, feedbackFor, err)
		return false
	}

	return !recorded
}

func latestActiveTreatmentPlan(dataAPI api.DataAPI, patientID int64) (*common.TreatmentPlan, error) {
	// Only show the feedback prompt if the patient has viewed the latest active treatment plan
	tps, err := dataAPI.GetActiveTreatmentPlansForPatient(patientID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get active treatment plans for patient %d: %s", patientID, err)
	}
	if len(tps) == 0 {
		return nil, nil
	}

	// Make sure latest treatment plan has been viewed
	var latest *common.TreatmentPlan
	for _, tp := range tps {
		if tp.SentDate != nil && (latest == nil || tp.SentDate.After(*latest.SentDate)) {
			latest = tp
		}
	}

	// Shouldn't happen but handle anyway (SentDate could be nil for some odd reason)
	if latest == nil {
		golog.Warningf("All active treatment plans have nil SentDate")
		return nil, nil
	}

	return latest, nil
}
