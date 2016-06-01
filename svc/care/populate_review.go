package care

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/svc/layout"
)

const (
	textReplacementIdentifier = "XXX"
)

// PopulateVisitReview returns a json representation of a visit in review form for the provider.
func PopulateVisitReview(intake *layout.Intake, review *visitreview.SectionListView, answers map[string]*Answer, visit *Visit) ([]byte, error) {
	context := visitreview.NewViewContext(nil)

	if err := populateAlerts(answers, intake, context); err != nil {
		return nil, errors.Trace(err)
	}

	// go through each question and populate context
	for _, section := range intake.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				answer := answers[question.ID]

				switch question.Type {
				case layout.QuestionTypeAutoComplete:
					if err := builderQuestionAutocomplete(question, answer, context); err != nil {
						return nil, errors.Trace(err)
					}
				case layout.QuestionTypeFreeText, layout.QuestionTypeSingleEntry:
					if err := builderQuestionFreeText(question, answer, context); err != nil {
						return nil, errors.Trace(err)
					}
				case layout.QuestionTypeMultipleChoice:
					if question.SubQuestionsConfig != nil {
						if err := builderQuestionWithSubanswers(question, answer, context); err != nil {
							return nil, errors.Trace(err)
						}
						continue
					}
					if err := builderQuestionWithOptions(question, answer, context); err != nil {
						return nil, errors.Trace(err)
					}
				case layout.QuestionTypeMediaSection:
					if err := builderQuestionWithMediaSlots(question, answer, context); err != nil {
						return nil, errors.Trace(err)
					}
				case layout.QuestionTypeSegmentedControl, layout.QuestionTypeSingleSelect:
					if err := builderQuestionWithSingleResponse(question, answer, context); err != nil {
						return nil, errors.Trace(err)
					}
				default:
					return nil, errors.Trace(fmt.Errorf("unknown question type (%s) for %s", question.Type, question.ID))
				}
			}
		}
	}

	renderedView, err := review.Render(context)
	if err != nil {
		return nil, errors.Trace(err)
	}

	reviewJSONData, err := json.Marshal(renderedView)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return reviewJSONData, nil
}

func populateAlerts(answers map[string]*Answer, intake *layout.Intake, context *visitreview.ViewContext) error {

	var alerts []string
	for _, question := range intake.Questions() {
		answer, ok := answers[question.ID]
		if !ok {
			// skip any unanswered questions
			continue
		}

		if question.ToAlert != nil && *question.ToAlert {

			switch question.Type {
			case layout.QuestionTypeAutoComplete:
				{
					// populate the answers to call out in the alert
					enteredAnswers := make([]string, len(answer.GetAutocomplete().Items))
					for i, item := range answer.GetAutocomplete().Items {
						enteredAnswers[i] = item.Answer
					}
					alerts = append(alerts, strings.Replace(question.AlertFormattedText, textReplacementIdentifier, join(enteredAnswers), -1))
				}
			case layout.QuestionTypeMultipleChoice:
				{
					selectedAnswers := make([]string, 0, len(question.PotentialAnswers))

					// go through all the potential answers of the question to identify the
					// ones that need to be alerted on
					for _, potentialAnswer := range question.PotentialAnswers {
						for _, patientAnswer := range answer.GetMultipleChoice().SelectedAnswers {
							if patientAnswer.ID == potentialAnswer.ID && potentialAnswer.ToAlert != nil && *potentialAnswer.ToAlert {
								if potentialAnswer.Summary != "" {
									selectedAnswers = append(selectedAnswers, potentialAnswer.Summary)
								} else {
									selectedAnswers = append(selectedAnswers, potentialAnswer.Answer)
								}
								break
							}
						}
					}
					// its possible that the patient selected an answer that need not be alerted on
					if len(selectedAnswers) > 0 {
						alerts = append(alerts, strings.Replace(question.AlertFormattedText, textReplacementIdentifier, join(selectedAnswers), -1))
					}
				}
			case layout.QuestionTypeSingleSelect:
				{
					for _, potentialAnswer := range question.PotentialAnswers {
						if potentialAnswer.ID == answer.GetSingleSelect().SelectedAnswer.ID {
							if potentialAnswer.ToAlert != nil && *potentialAnswer.ToAlert {
								text := potentialAnswer.Summary
								if text == "" {
									text = potentialAnswer.Answer
								}
								alerts = append(alerts, strings.Replace(question.AlertFormattedText, textReplacementIdentifier, text, -1))
								break
							}
						}
					}
				}
			default:
				return fmt.Errorf("cannot handle alerts for question (%s) of type %s", question.ID, question.Type)
			}
		}
	}

	if len(alerts) > 0 {
		context.Set("visit_alerts", alerts)
	} else {
		context.Set(visitreview.EmptyStateTextKey("visit_alerts"), "No alerts")
	}

	return nil
}

type buildContextFunc func(*layout.Question, *Answer, *visitreview.ViewContext) error

func builderQuestionWithOptions(question *layout.Question, answer *Answer, context *visitreview.ViewContext) error {
	if answer == nil {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	checkeUncheckedItems := make([]visitreview.CheckedUncheckedData, 0, len(question.PotentialAnswers))
	// in the event that there are free text entries for an answer selection,
	// populate them and show them next to the option selected, separated by commas
	var otherTextEntries []string
	for _, option := range question.PotentialAnswers {
		answerSelected := false
		text := option.Answer
		otherTextEntries = otherTextEntries[:0]
		for _, selectedAnswer := range answer.GetMultipleChoice().SelectedAnswers {
			if selectedAnswer.ID == option.ID {
				answerSelected = true
				if selectedAnswer.FreeText != "" {
					otherTextEntries = append(otherTextEntries, selectedAnswer.FreeText)
				}
			}
		}

		if len(otherTextEntries) > 0 {
			text += " - " + strings.Join(otherTextEntries, ",")
		}

		checkeUncheckedItems = append(checkeUncheckedItems, visitreview.CheckedUncheckedData{
			Value:     text,
			IsChecked: answerSelected,
		})
	}

	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.AnswersKey(question.ID), checkeUncheckedItems)
	return nil
}

func builderQuestionWithSingleResponse(question *layout.Question, answer *Answer, context *visitreview.ViewContext) error {
	if answer == nil {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	var text string
	switch question.Type {
	case layout.QuestionTypeSingleSelect:
		for _, option := range question.PotentialAnswers {
			if option.ID == answer.GetSingleSelect().SelectedAnswer.ID {
				if option.Summary != "" {
					text = option.Summary
				} else {
					text = option.Answer
				}
				if answer.GetSingleSelect().SelectedAnswer.FreeText != "" {
					text = text + " - " + answer.GetSingleSelect().SelectedAnswer.FreeText
				}
			}
		}
	case layout.QuestionTypeSegmentedControl:
		text = answer.GetSegmentedControl().SelectedAnswer.FreeText
		if text == "" {
			for _, option := range question.PotentialAnswers {
				if option.ID == answer.GetSegmentedControl().SelectedAnswer.ID {
					text = option.Summary
					if text == "" {
						text = option.Answer
					}
				}
			}
		}
	}

	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.AnswersKey(question.ID), text)
	return nil
}

func builderQuestionFreeText(question *layout.Question, answer *Answer, context *visitreview.ViewContext) error {
	if answer == nil {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	var text string
	if question.Type == layout.QuestionTypeFreeText {
		text = answer.GetFreeText().FreeText
	} else if question.Type == layout.QuestionTypeSingleEntry {
		text = answer.GetSingleEntry().FreeText
	}

	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.AnswersKey(question.ID), text)
	return nil
}

func builderQuestionAutocomplete(question *layout.Question, answer *Answer, context *visitreview.ViewContext) error {
	if answer == nil {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	subquestions := question.SubQuestions()
	data := make([]visitreview.TitleSubItemsDescriptionContentData, len(answer.GetAutocomplete().Items))
	for i, item := range answer.GetAutocomplete().Items {
		items := make([]*visitreview.DescriptionContentData, 0, len(item.SubAnswers))
		for _, subquestion := range subquestions {
			subanswer, ok := item.SubAnswers[subquestion.ID]
			if !ok {
				continue
			}
			content, err := contentForSubanswer(subquestion, subanswer)
			if err != nil {
				return errors.Trace(err)
			}
			items = append(items, &visitreview.DescriptionContentData{
				Description: subquestion.Summary,
				Content:     content,
			})
		}

		data[i] = visitreview.TitleSubItemsDescriptionContentData{
			Title:    item.Answer,
			SubItems: items,
		}
	}

	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.AnswersKey(question.ID), data)
	return nil
}

func builderQuestionWithSubanswers(question *layout.Question, answer *Answer, context *visitreview.ViewContext) error {

	if answer == nil {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	subquestions := question.SubQuestions()
	data := make([]visitreview.TitleSubItemsDescriptionContentData, len(answer.GetMultipleChoice().SelectedAnswers))
	for i, selectedAnswer := range answer.GetMultipleChoice().SelectedAnswers {
		items := make([]*visitreview.DescriptionContentData, 0, len(selectedAnswer.SubAnswers))
		for _, subquestion := range subquestions {
			subanswer, ok := selectedAnswer.SubAnswers[subquestion.ID]
			if !ok {
				continue
			}
			content, err := contentForSubanswer(subquestion, subanswer)
			if err != nil {
				return errors.Trace(err)
			}
			items = append(items, &visitreview.DescriptionContentData{
				Description: subquestion.Summary,
				Content:     content,
			})
		}

		// title is either a user-defined entry or the potential answer
		title := selectedAnswer.FreeText
		if title == "" {
			for _, option := range question.PotentialAnswers {
				if option.ID == selectedAnswer.ID {
					title = option.Answer
				}
			}
		}

		data[i] = visitreview.TitleSubItemsDescriptionContentData{
			Title:    title,
			SubItems: items,
		}
	}

	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.AnswersKey(question.ID), data)
	return nil
}

func contentForSubanswer(question *layout.Question, answer *Answer) (string, error) {
	switch question.Type {
	case layout.QuestionTypeFreeText:
		return answer.GetFreeText().FreeText, nil
	case layout.QuestionTypeSingleEntry:
		return answer.GetSingleEntry().FreeText, nil
	case layout.QuestionTypeSegmentedControl:
		if freeText := answer.GetSegmentedControl().SelectedAnswer.FreeText; freeText != "" {
			return freeText, nil
		}
		for _, option := range question.PotentialAnswers {
			if option.ID == answer.GetSegmentedControl().SelectedAnswer.ID {
				if option.Summary != "" {
					return option.Summary, nil
				}
				return option.Answer, nil
			}
		}
	case layout.QuestionTypeSingleSelect:
		if freeText := answer.GetSingleSelect().SelectedAnswer.FreeText; freeText != "" {
			return freeText, nil
		}
		for _, option := range question.PotentialAnswers {
			if option.ID == answer.GetSingleSelect().SelectedAnswer.ID {
				if option.Summary != "" {
					return option.Summary, nil
				}
				return option.Answer, nil
			}
		}
	case layout.QuestionTypeAutoComplete:
		if question.SubQuestionsConfig != nil {
			return "", errors.Trace(fmt.Errorf("subquestion %s has subquestions which is not supported for review rendering", question.ID))
		}
		entries := make([]string, 0, len(answer.GetAutocomplete().Items))
		for _, item := range answer.GetAutocomplete().Items {
			entries = append(entries, item.Answer)
		}
		return strings.Join(entries, ","), nil
	case layout.QuestionTypeMultipleChoice:
		if question.SubQuestionsConfig != nil {
			return "", errors.Trace(fmt.Errorf("subquestion %s has subquestions which is not supported for review rendering", question.ID))
		}

		entries := make([]string, 0, len(answer.GetMultipleChoice().SelectedAnswers))
		for _, selectedAnswer := range answer.GetMultipleChoice().SelectedAnswers {
			if selectedAnswer.FreeText != "" {
				entries = append(entries, selectedAnswer.FreeText)
				continue
			}

			for _, option := range question.PotentialAnswers {
				if option.ID == selectedAnswer.ID {
					if option.Summary != "" {
						entries = append(entries, option.Summary)
					} else {
						entries = append(entries, option.Answer)
					}
				}
			}
		}

		return strings.Join(entries, ","), nil

	case layout.QuestionTypeMediaSection:
		return "", errors.Trace(fmt.Errorf("subquestion %s has mediaquestion format which is not supported for review rendering", question.ID))
	}

	return "", errors.Trace(fmt.Errorf("unsupported subquestion %s of type %s", question.ID, question.Type))
}

func builderQuestionWithMediaSlots(question *layout.Question, answer *Answer, context *visitreview.ViewContext) error {
	if answer == nil {
		return nil
	}

	items := make([]visitreview.TitleMediaListData, 0, len(answer.GetMediaSection().Sections))
	for _, section := range answer.GetMediaSection().Sections {
		item := visitreview.TitleMediaListData{
			Title: section.Name,
			Media: make([]visitreview.MediaData, len(section.Slots)),
		}

		for i, slot := range section.Slots {
			item.Media[i] = visitreview.MediaData{
				Title:   slot.Name,
				MediaID: slot.MediaID,
				URL:     slot.URL,
				Type:    slot.Type,
				// TODO: populate real URL
			}
		}
		items = append(items, item)
	}

	context.Set(visitreview.MediaKey(question.ID), items)
	return nil
}

func populateEmptyStateTextIfPresent(question *layout.Question, context *visitreview.ViewContext) {
	if question.AdditionalFields == nil || question.AdditionalFields.EmptyStateText == "" {
		return
	}

	context.Set(visitreview.QuestionSummaryKey(question.ID), question.Summary)
	context.Set(visitreview.EmptyStateTextKey(question.ID), question.AdditionalFields.EmptyStateText)
}

func join(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}

	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}
