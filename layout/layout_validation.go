package layout

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/info_intake"
)

type errorList []string

func (e errorList) Error() string {
	return "layout.validate: " + strings.Join([]string(e), ", ")
}

func validateQuestion(que *info_intake.Question, path string, errors errorList) {
	if que.QuestionTag == "" {
		errors = append(errors, fmt.Sprintf("%s missing 'question'", path))
	}
	switch que.QuestionType {
	case info_intake.QUESTION_TYPE_MULTIPLE_CHOICE,
		info_intake.QUESTION_TYPE_SINGLE_SELECT,
		info_intake.QUESTION_TYPE_SEGMENTED_CONTROL:
		if len(que.PotentialAnswers) == 0 {
			errors = append(errors, fmt.Sprintf("%s missing potential answers", path))
		}
	case info_intake.QUESTION_TYPE_PHOTO_SECTION:
		if len(que.PhotoSlots) == 0 {
			errors = append(errors, fmt.Sprintf("%s missing photo slots", path))
		}
	case info_intake.QUESTION_TYPE_FREE_TEXT,
		info_intake.QUESTION_TYPE_AUTOCOMPLETE:
		if len(que.PotentialAnswers) != 0 {
			errors = append(errors, fmt.Sprintf("%s should not have potential answers", path))
		}
	}
	if c := que.SubQuestionsConfig; c != nil {
		for i, q := range c.Questions {
			validateQuestion(q, fmt.Sprintf("%s.subquestion[%d]", path, i), errors)
		}
	}
	if que.ConditionBlock != nil {
		switch que.ConditionBlock.OperationTag {
		case "":
			errors = append(errors, fmt.Sprintf("%s missing op in condition", path))
		case "answer_contains_any", "answer_equals":
			if que.ConditionBlock.QuestionTag == "" {
				errors = append(errors, fmt.Sprintf("%s missing question for '%s' condition", path, que.ConditionBlock.OperationTag))
			}
			if len(que.ConditionBlock.PotentialAnswersTags) == 0 {
				errors = append(errors, fmt.Sprintf("%s missing potential answers for '%s' condition", path, que.ConditionBlock.OperationTag))
			}
		case "gender_equals":
			if que.ConditionBlock.GenderField == "" {
				errors = append(errors, fmt.Sprintf("%s missing gender for '%s' condition", path, que.ConditionBlock.OperationTag))
			}
		default:
			errors = append(errors, fmt.Sprintf("%s unknown condition op '%s'", path, que.ConditionBlock.OperationTag))
		}
	}
}

func validatePatientLayout(layout *info_intake.InfoIntakeLayout) error {
	var errors errorList
	if len(layout.Sections) == 0 {
		errors = append(errors, "layout contains no sections")
	}
	if layout.PathwayTag == "" {
		errors = append(errors, "pathway tag not set")
	}
	for secIdx, sec := range layout.Sections {
		path := fmt.Sprintf("section[%d]", secIdx)
		if sec.SectionTag == "" {
			errors = append(errors, fmt.Sprintf("%s missing 'section'", path))
		}
		if len(sec.Screens) == 0 {
			errors = append(errors, fmt.Sprintf("%s has no screens", path))
		}
		for scrIdx, scr := range sec.Screens {

			if scr.ScreenType == "screen_type_pharmacy" {
				continue
			}

			path = fmt.Sprintf("%s.screen[%d]", path, scrIdx)
			if len(scr.Questions) == 0 {
				errors = append(errors, fmt.Sprintf("%s has no questions", path))
			}
			for queIdx, que := range scr.Questions {
				validateQuestion(que, fmt.Sprintf("%s.question[%d]", path, queIdx), errors)
			}
		}
	}
	if len(errors) != 0 {
		return errors
	}
	return nil
}

// Find all values that are strings that start with "q_" which represent a question
func questionMap(in interface{}, out map[string]bool) {
	switch v := in.(type) {
	case string:
		if strings.HasPrefix(v, "q_") {
			if idx := strings.IndexByte(v, ':'); idx > 0 {
				out[v[:idx]] = true
			}
		}
	case []interface{}:
		for _, v2 := range v {
			questionMap(v2, out)
		}
	case map[string]interface{}:
		for _, v2 := range v {
			questionMap(v2, out)
		}
	}
}

func reviewContext(patientLayout *info_intake.InfoIntakeLayout) (map[string]interface{}, error) {
	context := make(map[string]interface{})
	context["patient_visit_alerts"] = []string{"ALERT"}
	context["visit_message"] = "message"
	for _, sec := range patientLayout.Sections {
		if len(sec.Questions) != 0 {
			return nil, fmt.Errorf("Don't support questions in a section outside of a screen")
		}
		for _, scr := range sec.Screens {
			for _, que := range scr.Questions {
				switch que.QuestionType {
				case info_intake.QUESTION_TYPE_PHOTO_SECTION:
					photoList := make([]info_intake.TitlePhotoListData, len(que.PhotoSlots))
					for i, slot := range que.PhotoSlots {
						photoList[i] = info_intake.TitlePhotoListData{
							Title:  slot.Name,
							Photos: []info_intake.PhotoData{},
						}
					}
					context["patient_visit_photos"] = photoList
				case info_intake.QUESTION_TYPE_SINGLE_SELECT,
					info_intake.QUESTION_TYPE_SINGLE_ENTRY,
					info_intake.QUESTION_TYPE_FREE_TEXT:

					context[que.QuestionTag+":question_summary"] = "Summary"
					context[que.QuestionTag+":answers"] = "Answer"
				case info_intake.QUESTION_TYPE_MULTIPLE_CHOICE:
					if sub := que.SubQuestionsConfig; sub != nil {
						data := []info_intake.TitleSubItemsDescriptionContentData{
							info_intake.TitleSubItemsDescriptionContentData{
								Title: "Title",
								SubItems: []*info_intake.DescriptionContentData{
									&info_intake.DescriptionContentData{
										Description: "Description",
										Content:     "Content",
									},
								},
							},
						}
						context[que.QuestionTag+":question_summary"] = "Summary"
						context[que.QuestionTag+":answers"] = data
					} else {
						context[que.QuestionTag+":question_summary"] = "Summary"
						context[que.QuestionTag+":answers"] = []info_intake.CheckedUncheckedData{
							{Value: "Value", IsChecked: true},
						}
					}
				case info_intake.QUESTION_TYPE_AUTOCOMPLETE:
					data := []info_intake.TitleSubItemsDescriptionContentData{
						info_intake.TitleSubItemsDescriptionContentData{
							Title: "Title",
							SubItems: []*info_intake.DescriptionContentData{
								&info_intake.DescriptionContentData{
									Description: "Description",
									Content:     "Content",
								},
							},
						},
					}
					context[que.QuestionTag+":question_summary"] = "Summary"
					context[que.QuestionTag+":answers"] = data
				default:
					return nil, fmt.Errorf("Unknown question type '%s'", que.QuestionType)
				}
			}
		}
	}
	return context, nil
}

func compareQuestions(intakeLayout *info_intake.InfoIntakeLayout, reviewJS map[string]interface{}) error {
	intakeQuestions := map[string]bool{}
	conditionQuestions := map[string]bool{}
	for _, sec := range intakeLayout.Sections {
		if len(sec.Questions) != 0 {
			return fmt.Errorf("Questions in a section outside of a screen unsupported")
		}
		for _, scr := range sec.Screens {
			if scr.ScreenType == "screen_type_photo" {
				// Ignore photo sections since the question tags aren't used in the
				// same way that other questions are.
				continue
			}
			for _, que := range scr.Questions {
				intakeQuestions[que.QuestionTag] = true
				if con := que.ConditionBlock; con != nil {
					conditionQuestions[con.QuestionTag] = true
				}
			}
		}
	}

	reviewQuestions := map[string]bool{}
	questionMap(reviewJS, reviewQuestions)

	for q := range intakeQuestions {
		if !reviewQuestions[q] {
			// It's ok if the question doesn't show up in the review layout
			// if it's used in a condition.
			if !conditionQuestions[q] {
				return fmt.Errorf("Question '%s' in intake but not in review layout", q)
			}
		}
		delete(reviewQuestions, q)
	}
	if len(reviewQuestions) != 0 {
		s := make([]string, 0, len(reviewQuestions))
		for q := range reviewQuestions {
			s = append(s, q)
		}
		return fmt.Errorf("Question(s) '%s' in review layout but not in intake", strings.Join(s, ","))
	}

	return nil
}
