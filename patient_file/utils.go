package patient_file

import (
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/media"
)

// This interface is used to populate the ViewContext with data pertaining to a single question
type patientQAViewContextPopulator interface {
	populateViewContextWithPatientQA(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext) error
}

var patientQAPopulators map[string]patientQAViewContextPopulator = make(map[string]patientQAViewContextPopulator, 0)

func init() {
	patientQAPopulators[info_intake.QuestionTypeAutocomplete] = qaViewContextPopulator(populateAnswersForQuestionsWithSubanswers)
	patientQAPopulators[info_intake.QuestionTypeMultipleChoice] = qaViewContextPopulator(populateMultipleChoiceAnswers)
	patientQAPopulators[info_intake.QuestionTypeSingleEntry] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QuestionTypeFreeText] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QuestionTypeSingleSelect] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QuestionTypeSegmentedControl] = qaViewContextPopulator(populateSingleEntryAnswers)
	patientQAPopulators[info_intake.QuestionTypePhotoSection] = qaViewContextPopulator(populatePatientPhotos)
	patientQAPopulators[info_intake.QuestionTypePhoto] = qaViewContextPopulator(populatePatientPhotos)
}

type qaViewContextPopulator func([]common.Answer, *info_intake.Question, *common.ViewContext) error

func (q qaViewContextPopulator) populateViewContextWithPatientQA(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext) error {
	return q(patientAnswers, question, context)
}

func populateMultipleChoiceAnswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	// if we are dealing with a question that has subquestions defined, populate the context appropriately
	if question.SubQuestionsConfig != nil && (len(question.SubQuestionsConfig.Screens) > 0 || len(question.SubQuestionsConfig.Questions) > 0) {
		return populateAnswersForQuestionsWithSubanswers(patientAnswers, question, context)
	}

	checkedUncheckedItems := make([]info_intake.CheckedUncheckedData, 0)
	var otherTextEntries []string
	for _, potentialAnswer := range question.PotentialAnswers {
		answerSelected := false
		answerText := potentialAnswer.Answer
		otherTextEntries = otherTextEntries[:0]
		for _, patientAnswer := range patientAnswers {
			pAnswer := patientAnswer.(*common.AnswerIntake)
			if pAnswer.PotentialAnswerID.Int64() == potentialAnswer.AnswerID {
				answerSelected = true
				if pAnswer.AnswerText != "" {
					otherTextEntries = append(otherTextEntries, pAnswer.AnswerText)
				}
			}
		}

		if len(otherTextEntries) > 0 {
			answerText = fmt.Sprintf("%s - %s", answerText, strings.Join(otherTextEntries, ", "))
		}

		checkedUncheckedItems = append(checkedUncheckedItems, info_intake.CheckedUncheckedData{
			Value:     answerText,
			IsChecked: answerSelected,
		})
	}

	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:answers", question.QuestionTag), checkedUncheckedItems)
	return nil
}

func populateSingleEntryAnswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	if len(patientAnswers) > 1 {
		return errors.Trace(fmt.Errorf("Expected just one answer for question %s instead we have  %d", question.QuestionTag, len(patientAnswers)))
	}

	pAnswer := patientAnswers[0].(*common.AnswerIntake)
	answer := pAnswer.AnswerText
	if answer == "" {
		answer = pAnswer.AnswerSummary
	}
	if answer == "" {
		answer = pAnswer.PotentialAnswer
	}

	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:answers", question.QuestionTag), answer)
	return nil
}

func populateAnswersForQuestionsWithSubanswers(patientAnswers []common.Answer, question *info_intake.Question, context *common.ViewContext) error {
	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	// creating a mapping of id to subquestion
	qMapping := make(map[int64]*info_intake.Question)
	if question.SubQuestionsConfig != nil {
		for _, subQuestion := range question.SubQuestionsConfig.Questions {
			qMapping[subQuestion.QuestionID] = subQuestion
		}

		for _, screen := range question.SubQuestionsConfig.Screens {
			for _, subQuestion := range screen.Questions {
				qMapping[subQuestion.QuestionID] = subQuestion
			}
		}
	}

	data := make([]info_intake.TitleSubItemsDescriptionContentData, len(patientAnswers))
	for i, patientAnswer := range patientAnswers {
		pAnswer := patientAnswer.(*common.AnswerIntake)
		items := make([]*info_intake.DescriptionContentData, 0, len(pAnswer.SubAnswers))
		for _, subAnswer := range pAnswer.SubAnswers {
			// user-entered answer gets priority, then any summary for an answer, followed by the potential answer
			// if it exists
			if subAnswer.AnswerText != "" {
				items = append(items, &info_intake.DescriptionContentData{
					Description: qMapping[subAnswer.QuestionID.Int64()].QuestionSummary,
					Content:     subAnswer.AnswerText,
				})
			} else if subAnswer.AnswerSummary != "" {
				items = append(items, &info_intake.DescriptionContentData{
					Description: qMapping[subAnswer.QuestionID.Int64()].QuestionSummary,
					Content:     subAnswer.AnswerSummary,
				})
			} else if subAnswer.PotentialAnswer != "" {
				items = append(items, &info_intake.DescriptionContentData{
					Description: qMapping[subAnswer.QuestionID.Int64()].QuestionSummary,
					Content:     subAnswer.PotentialAnswer,
				})
			}
		}

		// title is either a user-defined entry or the potential answer
		title := pAnswer.AnswerText
		if title == "" {
			title = pAnswer.PotentialAnswer
		}

		data[i] = info_intake.TitleSubItemsDescriptionContentData{
			Title:    title,
			SubItems: items,
		}
	}
	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:answers", question.QuestionTag), data)
	return nil
}

// if there are no patient answers for this question,
// check if the empty state text is specified in the additional fields
// of the question
func populateEmptyStateTextIfPresent(question *info_intake.Question, context *common.ViewContext) {
	emptyStateText, ok := question.AdditionalFields["empty_state_text"]
	if !ok {
		return
	}

	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:empty_state_text", question.QuestionTag), emptyStateText)
}

func populatePatientPhotos(answeredPhotoSections []common.Answer, question *info_intake.Question, context *common.ViewContext) error {
	if len(answeredPhotoSections) == 0 {
		return nil
	}

	var items []info_intake.TitlePhotoListData
	// continue to populate the global patient_visit_photos
	// key for backwards compatibility, given that acne related
	// doctor reviews expect this key to exist.
	photoData, ok := context.Get("patient_visit_photos")

	if !ok || photoData == nil {
		items = make([]info_intake.TitlePhotoListData, 0, len(answeredPhotoSections))
	} else {
		items = photoData.([]info_intake.TitlePhotoListData)
	}

	// keep track of the question specific items so that we can create a key to link to
	// photos pertaining to a question
	questionSpecificItems := make([]info_intake.TitlePhotoListData, 0, len(answeredPhotoSections))

	for _, photoSection := range answeredPhotoSections {
		pIntakeSection := photoSection.(*common.PhotoIntakeSection)
		item := info_intake.TitlePhotoListData{
			Title:  pIntakeSection.Name,
			Photos: make([]info_intake.PhotoData, len(pIntakeSection.Photos)),
		}

		for i, photoIntakeSlot := range pIntakeSection.Photos {
			item.Photos[i] = info_intake.PhotoData{
				Title:    photoIntakeSlot.Name,
				PhotoID:  photoIntakeSlot.PhotoID,
				PhotoURL: photoIntakeSlot.PhotoURL,
			}
		}
		items = append(items, item)
		questionSpecificItems = append(questionSpecificItems, item)
	}

	context.Set("patient_visit_photos", items)
	context.Set(question.QuestionTag+":photos", questionSpecificItems)
	return nil
}

func buildContext(
	dataAPI api.DataAPI,
	mediaStore *media.Store,
	expirationDuration time.Duration,
	visitLayout *info_intake.InfoIntakeLayout,
	pat *common.Patient,
	visit *common.PatientVisit) (*common.ViewContext, error) {

	context, err := populateContextForRenderingLayout(
		visitLayout.Answers(),
		visitLayout.Questions(),
		dataAPI,
		mediaStore,
		expirationDuration,
		pat,
		visit.ID.Int64())
	return context, errors.Trace(err)
}

func populateContextForRenderingLayout(
	answers map[int64][]common.Answer,
	questions []*info_intake.Question,
	dataAPI api.DataAPI,
	mediaStore *media.Store,
	expirationDuration time.Duration,
	patient *common.Patient,
	patientVisitID int64,
) (*common.ViewContext, error) {
	context := common.NewViewContext(nil)

	// populate alerts
	alerts, err := dataAPI.AlertsForVisit(patientVisitID)
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(alerts) > 0 {
		alertsArray := make([]string, len(alerts))
		for i, alert := range alerts {
			alertsArray[i] = alert.Message
		}
		context.Set("patient_visit_alerts", alertsArray)
	} else {
		context.Set("patient_visit_alerts:empty_state_text", "No alerts")
	}

	// populate message for patient visit if one exists
	message, err := dataAPI.GetMessageForPatientVisit(patientVisitID)
	if err != nil && !api.IsErrNotFound(err) {
		return nil, errors.Trace(err)
	}
	if message != "" {
		context.Set("q_anything_else_acne:answers", message)
		context.Set("visit_message", message)
	} else {
		context.Set("visit_message:empty_state_text", "Patient did not specify")
	}

	// only populate parent info if patient is under 18 and has parental consent
	if patient.IsUnder18() && patient.HasParentalConsent {
		if err := populateParentInfo(dataAPI, mediaStore, expirationDuration, patient, context); err != nil {
			return nil, errors.Trace(err)
		}
	}

	// go through each question
	for _, question := range questions {
		contextPopulator, ok := patientQAPopulators[question.QuestionType]
		if !ok {
			return nil, errors.Trace(fmt.Errorf("Context populator not found for question with type %s", question.QuestionType))
		}

		if err := contextPopulator.populateViewContextWithPatientQA(answers[question.QuestionID], question, context); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return context, nil
}

func populateParentInfo(
	dataAPI api.DataAPI,
	mediaStore *media.Store,
	expirationDuration time.Duration,
	patient *common.Patient,
	context *common.ViewContext,
) error {
	consents, err := dataAPI.ParentalConsent(patient.ID)
	if err != nil {
		return errors.Trace(err)
	}

	if len(consents) == 0 {
		return nil
	}

	// Find the parent that actually gave consent (should only be one)
	var consent *common.ParentalConsent
	for _, c := range consents {
		if consent == nil || c.Consented {
			consent = c
		}
	}

	par := conc.NewParallel()

	// get parent patient info
	var parentPatient *common.Patient
	par.Go(func() error {
		var err error
		parentPatient, err = dataAPI.GetPatientFromID(consent.ParentPatientID)
		return errors.Trace(err)
	})

	var proof *api.ParentalConsentProof
	par.Go(func() error {
		var err error
		proof, err = dataAPI.ParentConsentProof(consent.ParentPatientID)
		return errors.Trace(err)
	})

	if err := par.Wait(); err != nil {
		return errors.Trace(err)
	}

	// indicate the fact that parent information is included
	context.Set("parent_information_included", true)

	// Parent name
	context.Set("parent_name:key", "Name")
	context.Set("parent_name:value", fmt.Sprintf("%s %s", parentPatient.FirstName, parentPatient.LastName))

	// Parent dob
	context.Set("parent_dob:key", "Date of Birth")
	context.Set("parent_dob:value", fmt.Sprintf("%02d/%02d/%d", parentPatient.DOB.Month, parentPatient.DOB.Day, parentPatient.DOB.Year))

	// Parent gender
	context.Set("parent_gender:key", "Gender")
	context.Set("parent_gender:value", strings.Title(parentPatient.Gender))

	// Parent relationship
	context.Set("parent_relationship:key", "Relationship")
	context.Set("parent_relationship:value", consent.Relationship)

	// Parent Photo ID
	photoSection := info_intake.TitlePhotoListData{
		Title:  "ID Verification",
		Photos: make([]info_intake.PhotoData, 0, 2),
	}

	// Include parent photo ids if present
	if proof.GovernmentIDPhotoID != nil {
		signedURL, err := mediaStore.SignedURL(*proof.GovernmentIDPhotoID, expirationDuration)
		if err != nil {
			return errors.Trace(err)
		}

		photoSection.Photos = append(photoSection.Photos, info_intake.PhotoData{
			Title:    "ID Verification",
			PhotoID:  *proof.GovernmentIDPhotoID,
			PhotoURL: signedURL,
		})
	}

	if proof.SelfiePhotoID != nil {
		signedURL, err := mediaStore.SignedURL(*proof.SelfiePhotoID, expirationDuration)
		if err != nil {
			return errors.Trace(err)
		}

		photoSection.Photos = append(photoSection.Photos, info_intake.PhotoData{
			Title:    "ID Verification",
			PhotoID:  *proof.SelfiePhotoID,
			PhotoURL: signedURL,
		})
	}
	context.Set("parent_photo_verification", []info_intake.TitlePhotoListData{photoSection})

	return nil
}
