package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/info_intake"
	"carefront/libs/pharmacy"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/SpruceHealth/mapstructure"
	"github.com/gorilla/schema"
)

type DoctorPatientVisitReviewHandler struct {
	DataApi                    api.DataAPI
	PharmacySearchService      pharmacy.PharmacySearchAPI
	LayoutStorageService       api.CloudStorageAPI
	PatientPhotoStorageService api.CloudStorageAPI
}

type DoctorPatientVisitReviewRequestBody struct {
	PatientVisitId  int64 `schema:"patient_visit_id"`
	TreatmentPlanId int64 `schema:"treatment_plan_id"`
}

type DoctorPatientVisitReviewResponse struct {
	Patient            *common.Patient        `json:"patient"`
	PatientVisit       *common.PatientVisit   `json:"patient_visit"`
	TreatmentPlanId    int64                  `json:"treatment_plan_id"`
	PatientVisitReview map[string]interface{} `json:"visit_review"`
}

var reviewTemplate = `{
  "type": "d_visit_review:sections_list",
  "sections": [
    {
      "title": "Photos",
      "type": "d_visit_review:standard_photo_section",
      "subsections": [
        {
          "type": "d_visit_review:standard_photo_subsection",
          "view": {
            "type": "d_visit_review:standard_photos_list",
            "content_config": {
              "key": "patient_visit_photos"
            }
          }
        }
      ]
    },
    {
      "title": "Medical History",
      "type": "d_visit_review:standard_section",
      "subsections": [
        {
          "title": "Alerts",
          "type": "d_visit_review:standard_subsection",
          "rows": [
            {
              "type": "d_visit_review:standard_one_column_row",
              "view": {
                "type": "d_visit_review:alert_labels_list",
                "content_config": {
                  "key": "patient_visit_alerts"
                },
                "empty_state_view": {
                  "type": "d_visit_review:empty_label",
                  "content_config": {
                    "key": "patient_visit_alerts:empty_state_text"
                  }
                }
              }
            }
          ]
        },
        {
          "title": "Medication",
          "type": "d_visit_review:standard_subsection",
          "rows": [
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_allergic_medication_entry:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:content_labels_list",
                "content_config": {
                  "key": "q_allergic_medication_entry:answers"
                },
                "empty_state_view": {
                  "type": "d_visit_review:empty_label",
                  "content_config": {
                    "key": "q_allergic_medication_entry:empty_state_text"
                  }
                }
              }
            },
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_current_medications_entry:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:title_subtitle_subitems_divided_items_list",
                "content_config": {
                  "key": "q_current_medications_entry:answers"
                },
                "empty_state_view": {
                  "type": "d_visit_review:empty_label",
                  "content_config": {
                    "key": "q_current_medications_entry:empty_state_text"
                  }
                }
              }
            }
          ]
        },
        {
          "title": "Past Medical Conditions",
          "type": "d_visit_review:standard_subsection",
          "rows": [
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_list_prev_skin_condition_diagnosis:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:content_labels_list",
                "content_config": {
                  "key": "q_list_prev_skin_condition_diagnosis:answers"
                },
                "empty_state_view": {
                  "type": "d_visit_review:empty_label",
                  "content_config": {
                    "key": "q_list_prev_skin_condition_diagnosis:empty_state_text"
                  }
                }
              }
            },
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_other_conditions_acne:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:check_x_items_list",
                "content_config": {
                  "key": "q_other_conditions_acne:answers"
                }
              }
            },
            {
              "type": "d_visit_review:standard_two_column_row",
              "content_config": {
                "condition": {
                  "op": "key_exists",
                  "key": "q_other_skin_condition_entry:question_summary"
                }
              },
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_other_skin_condition_entry:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:content_labels_list",
                "content_config": {
                  "key": "q_other_skin_condition_entry:answers"
                }
              }
            }
          ]
        },
        {
          "title": "Additional Information",
          "type": "d_visit_review:standard_subsection",
          "content_config": {
            "condition": {
              "op": "key_exists",
              "key": "q_pregnancy_planning:answers"
            }
          },
          "rows": [
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_pregnancy_planning:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:check_x_items_list",
                "content_config": {
                  "key": "q_pregnancy_planning:answers"
                }
              }
            }
          ]
        }
      ]
    },
    {
      "title": "History of Present Illness",
      "type": "d_visit_review:standard_section",
      "subsections": [
        {
          "title": "Symptoms Overview",
          "type": "d_visit_review:standard_subsection",
          "rows": [
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_onset_acne:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:content_labels_list",
                "content_config": {
                  "key": "q_onset_acne:answers"
                }
              }
            },
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_acne_location:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:check_x_items_list",
                "content_config": {
                  "key": "q_acne_location:answers"
                }
              }
            },
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_acne_symptoms:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:check_x_items_list",
                "content_config": {
                  "key": "q_acne_symptoms:answers"
                }
              }
            },
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_acne_worse:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:divided_views_list",
                "views": [
                  {
                    "type": "d_visit_review:content_labels_list",
                    "content_config": {
                      "key": "q_acne_worse:answers"
                    }
                  },
                  {
                    "type": "d_visit_review:title_subtitle_labels",
                    "content_config": {
                      "condition": {
                        "op": "key_exists",
                        "key": "q_changes_acne_worse:question_summary"
                      },
                      "title_key": "q_changes_acne_worse:question_summary",
                      "subtitle_key": "q_changes_acne_worse:answers"
                    },
                    "empty_state_view": {
                      "type": "d_visit_review:empty_title_subtitle_labels",
                      "content_config": {
                        "title_key": "q_changes_acne_worse:question_summary",
                        "subtitle_key": "q_changes_acne_worse:empty_state_text"
                      }
                    }
                  }
                ]
              }
            },
            {
              "type": "d_visit_review:standard_two_column_row",
              "content_config": {
                "condition": {
                  "op": "key_exists",
                  "key": "q_acne_worse_period:question_summary"
                }
              },
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_acne_worse_period:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:divided_views_list",
                "views": [
                  {
                    "type": "d_visit_review:content_labels_list",
                    "content_config": {
                      "key": "q_acne_worse_period:answers"
                    }
                  },
                  {
                    "type": "d_visit_review:title_subtitle_labels",
                    "content_config": {
                      "condition": {
                        "op": "key_exists",
                        "key": "q_periods_regular:question_summary"
                      },
                      "title_key": "q_periods_regular:question_summary",
                      "subtitle_key": "q_periods_regular:answers"
                    }
                  }
                ]
              }
            },
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_skin_description:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:content_labels_list",
                "content_config": {
                  "key": "q_skin_description:answers"
                }
              }
            }
          ]
        },
        {
          "title": "Prior Treatments",
          "type": "d_visit_review:standard_subsection",
          "rows": [
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_acne_prev_treatment_types:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:content_labels_list",
                "content_config": {
                  "key": "q_acne_prev_treatment_types:answers"
                }
              }
            },
            {
              "type": "d_visit_review:standard_two_column_row",
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_acne_prev_treatment_list:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:title_subtitle_subitems_divided_items_list",
                "content_config": {
                  "key": "q_acne_prev_treatment_list:answers"
                },
                "empty_state_view": {
                  "type": "d_visit_review:empty_label",
                  "content_config": {
                    "key": "q_acne_prev_treatment_list:empty_state_text"
                  }
                }
              }
            },
            {
              "type": "d_visit_review:standard_two_column_row",
              "content_config": {
                "condition": {
                  "op": "key_exists",
                  "key": "q_acne_prev_otc_treatment_list:question_summary"
                }
              },
              "left_view": {
                "type": "d_visit_review:title_labels_list",
                "content_config": {
                  "key": "q_acne_prev_otc_treatment_list:question_summary"
                }
              },
              "right_view": {
                "type": "d_visit_review:title_subtitle_subitems_divided_items_list",
                "content_config": {
                  "key": "q_acne_prev_otc_treatment_list:answers"
                },
                "empty_state_view": {
                  "type": "d_visit_review:empty_label",
                  "content_config": {
                    "key": "q_acne_prev_otc_treatment_list:empty_state_text"
                  }
                }
              }
            }
          ]
        },
        {
          "title": "Additional Information from Patient",
          "type": "d_visit_review:standard_subsection",
          "rows": [
            {
              "type": "d_visit_review:standard_one_column_row",
              "view": {
                "type": "d_visit_review:content_labels_list",
                "content_config": {
                  "key": "q_anything_else_acne:answers"
                },
                "empty_state_view": {
                  "type": "d_visit_review:empty_label",
                  "content_config": {
                    "key": "q_anything_else_acne:empty_state_text"
                  }
                }
              }
            }
          ]
        }
      ]
    }
  ]
}`

func (p *DoctorPatientVisitReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData DoctorPatientVisitReviewRequestBody
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisitId := requestData.PatientVisitId
	treatmentPlanId := requestData.TreatmentPlanId
	if err := ensureTreatmentPlanOrPatientVisitIdPresent(p.DataApi, treatmentPlanId, &patientVisitId); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientVisit, err := p.DataApi.GetPatientVisitFromId(patientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to get patient visit information from database based on provided patient visit id : "+err.Error())
		return
	}

	// ensure that the doctor is authorized to work on this case
	patientVisitReviewData, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(patientVisit.PatientVisitId.Int64(), GetContext(r).AccountId, p.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	// udpate the status of the case and the item in the doctor's queue
	if patientVisit.Status == api.CASE_STATUS_SUBMITTED {
		treatmentPlanId, err = p.DataApi.StartNewTreatmentPlanForPatientVisit(patientVisit.PatientId.Int64(), patientVisit.PatientVisitId.Int64(), patientVisitReviewData.DoctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the status of the visit to reviewing: "+err.Error())
			return
		}

		if err := p.DataApi.MarkPatientVisitAsOngoingInDoctorQueue(patientVisitReviewData.DoctorId, patientVisit.PatientVisitId.Int64()); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to update the item in the queue for the doctor that speaks to this patient visit: "+err.Error())
			return
		}

		if err := p.DataApi.RecordDoctorAssignmentToPatientVisit(patientVisit.PatientVisitId.Int64(), patientVisitReviewData.DoctorId); err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to assign the patient visit to this doctor: "+err.Error())
			return
		}
	} else {
		treatmentPlanId, err = p.DataApi.GetActiveTreatmentPlanForPatientVisit(patientVisitReviewData.DoctorId, patientVisit.PatientVisitId.Int64())
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment plan id for patient visit: "+err.Error())
			return
		}
	}

	patientVisitLayout, _, err := getClientLayoutForPatientVisit(patientVisitId, api.EN_LANGUAGE_ID, p.DataApi, p.LayoutStorageService)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient visit layout: "+err.Error())
		return
	}

	// get all questions presented to the patient in the patient visit layout
	questions := getQuestionsInPatientVisitLayout(patientVisitLayout)
	questionIds := getQuestionIdsInPatientVisitLayout(patientVisitLayout)

	// get all the answers the patient entered for the questions (note that there may not be an answer for every question)
	patientAnswersForQuestions, err := p.DataApi.GetAnswersForQuestionsBasedOnQuestionIds(questionIds, patientVisit.PatientId.Int64(), patientVisit.PatientVisitId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient answers for questions : "+err.Error())
		return
	}

	context, err := populateContextForRenderingLayout(patientAnswersForQuestions, questions, p.DataApi, p.PatientPhotoStorageService)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to populate context for rendering layout: "+err.Error())
		return
	}

	// TODO get the appropriate template to render here

	var jsonData map[string]interface{}
	err = json.Unmarshal([]byte(reviewTemplate), &jsonData)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unbale to unmarshal file contents into map[string]interface{}: "+err.Error())
	}

	sectionList := &DVisitReviewSectionListView{}
	decoderConfig := &mapstructure.DecoderConfig{
		Result:  sectionList,
		TagName: "json",
	}
	decoderConfig.SetRegistry(dVisitReviewViewTypeRegistry.Map())

	d, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create new decoder: "+err.Error())
		return
	}

	err = d.Decode(jsonData)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to parse template into structure: "+err.Error())
		return
	}

	renderedJsonData, err := sectionList.Render(context)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to render template into expected view layout for doctor visit review: "+err.Error())
		return
	}

	response := &DoctorPatientVisitReviewResponse{}
	response.PatientVisit = patientVisit
	patient, err := p.DataApi.GetPatientFromId(patientVisit.PatientId.Int64())
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patient based on id: "+err.Error())
		return
	}

	response.Patient = patient
	response.TreatmentPlanId = treatmentPlanId
	response.PatientVisitReview = renderedJsonData

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, response)
}

func populateContextForRenderingLayout(patientAnswersForQuestions map[int64][]*common.AnswerIntake, questions []*info_intake.Question, dataApi api.DataAPI, photoStorageService api.CloudStorageAPI) (common.ViewContext, error) {
	context := common.NewViewContext()

	populateAlerts(patientAnswersForQuestions, questions, context, dataApi)

	// go through each question
	for _, question := range questions {
		switch question.QuestionTypes[0] {

		case info_intake.QUESTION_TYPE_PHOTO, info_intake.QUESTION_TYPE_MULTIPLE_PHOTO, info_intake.QUESTION_TYPE_SINGLE_PHOTO:
			populatePhotos(patientAnswersForQuestions[question.QuestionId], context, photoStorageService)

		case info_intake.QUESTION_TYPE_AUTOCOMPLETE:
			populateDataForAnswerWithSubAnswers(patientAnswersForQuestions[question.QuestionId], question, context)

		case info_intake.QUESTION_TYPE_MULTIPLE_CHOICE:
			if err := populateCheckedUncheckedData(patientAnswersForQuestions[question.QuestionId], question, context, dataApi); err != nil {
				return nil, err
			}

		case info_intake.QUESTION_TYPE_SINGLE_ENTRY, info_intake.QUESTION_TYPE_FREE_TEXT, info_intake.QUESTION_TYPE_SINGLE_SELECT:
			if err := populateDataForSingleEntryAnswers(patientAnswersForQuestions[question.QuestionId], question, context); err != nil {
				return nil, err
			}
		}
	}

	return *context, nil
}

func populateAlerts(patientAnswers map[int64][]*common.AnswerIntake, questions []*info_intake.Question, context *common.ViewContext, dataApi api.DataAPI) error {

	questionIdToQuestion := make(map[int64]*info_intake.Question)
	for _, question := range questions {
		questionIdToQuestion[question.QuestionId] = question
	}

	alerts := make([]string, 0)
	// lets go over every answered question
	for questionId, answers := range patientAnswers {
		// check if the alert flag is set on the question
		question := questionIdToQuestion[questionId]
		if question.ToAlert {
			switch question.QuestionTypes[0] {

			case info_intake.QUESTION_TYPE_AUTOCOMPLETE:
				// populate the answers to call out in the alert
				enteredAnswers := make([]string, len(answers))
				for i, answer := range answers {

					answerText := answer.AnswerText

					if answerText == "" {
						answerText = answer.AnswerSummary
					}

					if answerText == "" {
						answerText = answer.PotentialAnswer
					}

					enteredAnswers[i] = answerText
				}
				if len(enteredAnswers) > 0 {
					alerts = append(alerts, fmt.Sprintf(question.AlertFormattedText, strings.Join(enteredAnswers, ", ")))
				}

			case info_intake.QUESTION_TYPE_MULTIPLE_CHOICE, info_intake.QUESTION_TYPE_SINGLE_SELECT:
				selectedAnswers := make([]string, 0)
				for _, potentialAnswer := range question.PotentialAnswers {
					for _, patientAnswer := range answers {
						// populate all the selected answers to show in the alert
						if patientAnswer.PotentialAnswerId.Int64() == potentialAnswer.AnswerId {
							if potentialAnswer.ToAlert {
								selectedAnswers = append(selectedAnswers, potentialAnswer.Answer)
								break
							}
						}
					}
				}
				if len(selectedAnswers) > 0 {
					alerts = append(alerts, fmt.Sprintf(question.AlertFormattedText, strings.Join(selectedAnswers, ", ")))
				}
			}
		}
	}

	if len(alerts) > 0 {
		context.Set("patient_visit_alerts", alerts)
	} else {
		context.Set("patient_visit_alerts:empty_state_text", "No alerts")
	}

	return nil
}

func populateCheckedUncheckedData(patientAnswers []*common.AnswerIntake, question *info_intake.Question, context *common.ViewContext, dataApi api.DataAPI) error {

	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	checkedUncheckedItems := make([]CheckedUncheckedData, len(question.PotentialAnswers))
	for i, potentialAnswer := range question.PotentialAnswers {
		answerSelected := false

		for _, patientAnswer := range patientAnswers {
			if patientAnswer.PotentialAnswerId.Int64() == potentialAnswer.AnswerId {
				answerSelected = true
			}
		}

		checkedUncheckedItems[i] = CheckedUncheckedData{
			Value:     potentialAnswer.Answer,
			IsChecked: answerSelected,
		}
	}

	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:answers", question.QuestionTag), checkedUncheckedItems)
	return nil
}

func populatePhotos(patientAnswers []*common.AnswerIntake, context *common.ViewContext, photoStorageService api.CloudStorageAPI) {
	var photos []PhotoData
	photoData, ok := context.Get("patient_visit_photos")

	if !ok || photoData == nil {
		photos = make([]PhotoData, 0)
	} else {
		photos = photoData.([]PhotoData)
	}

	for _, answerIntake := range patientAnswers {
		photos = append(photos, PhotoData{
			Title:    answerIntake.PotentialAnswer,
			PhotoUrl: GetSignedUrlForAnswer(answerIntake, photoStorageService),
		})
	}

	context.Set("patient_visit_photos", photos)
}

func populateDataForSingleEntryAnswers(patientAnswers []*common.AnswerIntake, question *info_intake.Question, context *common.ViewContext) error {

	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return nil
	}

	if len(patientAnswers) > 1 {
		return fmt.Errorf("Expected just one answer for question %s instead we have  %d", question.QuestionTag, len(patientAnswers))
	}

	answer := patientAnswers[0].AnswerText
	if answer == "" {
		answer = patientAnswers[0].AnswerSummary
	}
	if answer == "" {
		answer = patientAnswers[0].PotentialAnswer
	}

	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:answers", question.QuestionTag), answer)
	return nil
}

func populateDataForAnswerWithSubAnswers(patientAnswers []*common.AnswerIntake, question *info_intake.Question, context *common.ViewContext) {

	if len(patientAnswers) == 0 {
		populateEmptyStateTextIfPresent(question, context)
		return
	}

	data := make([]TitleSubtitleSubItemsData, len(patientAnswers))
	for i, patientAnswer := range patientAnswers {

		items := make([]string, len(patientAnswer.SubAnswers))
		for j, subAnswer := range patientAnswer.SubAnswers {
			if subAnswer.AnswerSummary != "" {
				items[j] = subAnswer.AnswerSummary
			} else {
				items[j] = subAnswer.PotentialAnswer
			}
		}

		data[i] = TitleSubtitleSubItemsData{
			Title:    patientAnswer.AnswerText,
			SubItems: items,
		}
	}
	context.Set(fmt.Sprintf("%s:question_summary", question.QuestionTag), question.QuestionSummary)
	context.Set(fmt.Sprintf("%s:answers", question.QuestionTag), data)
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
