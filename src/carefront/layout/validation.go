package layout

import (
	"carefront/info_intake"
	"fmt"
	"strings"
)

type errorList []string

func (e errorList) Error() string {
	return "layout.validate: " + strings.Join([]string(e), ", ")
}

func validatePatientLayout(layout *info_intake.InfoIntakeLayout) error {
	var errors errorList
	if len(layout.Sections) == 0 {
		errors = append(errors, "layout contains no sections")
	}
	if layout.HealthConditionTag == "" {
		errors = append(errors, "health condition tag not set")
	}
	for secIdx, sec := range layout.Sections {
		if sec.SectionTag == "" {
			errors = append(errors, fmt.Sprintf("section %d missing 'section'", secIdx))
		}
		if len(sec.Screens) == 0 {
			errors = append(errors, fmt.Sprintf("section %d has no screens", secIdx))
		}
		for scrIdx, scr := range sec.Screens {
			if len(scr.Questions) == 0 {
				errors = append(errors, fmt.Sprintf("screen %d in section %d has no questions", scrIdx, secIdx))
			}
			for queIdx, que := range scr.Questions {
				if que.QuestionTag == "" {
					errors = append(errors, fmt.Sprintf("question %d on screen %d in section %d missing 'question'", queIdx, scrIdx, secIdx))
				}
				if que.ConditionBlock != nil {
					switch que.ConditionBlock.OperationTag {
					case "":
						errors = append(errors, fmt.Sprintf("question %d on screen %d in section %d missing op in condition", queIdx, scrIdx, secIdx))
					case "answer_contains_any", "answer_equals":
						if que.ConditionBlock.QuestionTag == "" {
							errors = append(errors, fmt.Sprintf("question %d on screen %d in section %d missing question for '%s' condition", queIdx, scrIdx, secIdx, que.ConditionBlock.OperationTag))
						}
						if len(que.ConditionBlock.PotentialAnswersTags) == 0 {
							errors = append(errors, fmt.Sprintf("question %d on screen %d in section %d missing potential answers for '%s' condition", queIdx, scrIdx, secIdx, que.ConditionBlock.OperationTag))
						}
					case "gender_equals":
						if que.ConditionBlock.GenderField == "" {
							errors = append(errors, fmt.Sprintf("question %d on screen %d in section %d missing gender for '%s' condition", queIdx, scrIdx, secIdx, que.ConditionBlock.OperationTag))
						}
					default:
						errors = append(errors, fmt.Sprintf("question %d on screen %d in section %d unknown condition op '%s'", queIdx, scrIdx, secIdx, que.ConditionBlock.OperationTag))
					}
				}
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
	for _, sec := range patientLayout.Sections {
		if len(sec.Questions) != 0 {
			return nil, fmt.Errorf("Don't support questions in a section outside of a screen")
		}
		for _, scr := range sec.Screens {
			for _, que := range scr.Questions {
				switch que.QuestionType {
				case "q_type_photo_section":
					photoList := make([]info_intake.TitlePhotoListData, len(que.PhotoSlots))
					for i, slot := range que.PhotoSlots {
						photoList[i] = info_intake.TitlePhotoListData{
							Title:  slot.Name,
							Photos: []info_intake.PhotoData{},
						}
					}
					context["patient_visit_photos"] = photoList
				case "q_type_single_select", "q_type_single_entry", "q_type_free_text":
					context[que.QuestionTag+":question_summary"] = "Summary"
					context[que.QuestionTag+":answers"] = "Answer"
				case "q_type_multiple_choice":
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
				case "q_type_autocomplete":
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
