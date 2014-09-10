package demo

import "github.com/sprucehealth/backend/common"

const (
	signupPatientUrl         = "/v1/patient"
	updatePatientPharmacyUrl = "/v1/patient/pharmacy"
	patientVisitUrl          = "/v1/patient/visit"
	answerQuestionsUrl       = "/v1/patient/visit/answer"
	photoIntakeUrl           = "/v1/patient/visit/photo_answer"
	messagesUrl              = "/v1/case/messages"
	regimenUrl               = "/v1/doctor/visit/regimen"
	addTreatmentsUrl         = "/v1/doctor/visit/treatment/treatments"
	dVisitReviewUrl          = "/v1/doctor/visit/review"
	dFavoriteTPUrl           = "/v1/doctor/favorite_treatment_plans"
	dTPUrl                   = "/v1/doctor/treatment_plans"
	dAuthUrl                 = "/v1/doctor/authenticate"
	visitMessageUrl          = "/v1/patient/visit/message"
	demoPhotosBucketFormat   = "%s-carefront-demo"
)

var LocalServerURL = "http://127.0.0.1:8080"

type questionTag int

const (
	qAcneOnset questionTag = iota
	qAcneWorse
	qSkinPhotoComparison
	qAcneContributingFactors
	qAcneSymptoms
	qAcneWorsePeriod
	qPeriodsRegular
	qSkinDescription
	qAcnePrevTreatmentTypes
	qPregnancyPlanning
	qCurrentMedications
	qCurrentMedicationsEntry
	qLengthCurrentMedication
	qAllergicMedications
	qAllergicMedicationEntry
	qPrevSkinConditionDiagnosis
	qListPrevSkinConditionDiagnosis
	qOtherConditionsAcne
	qFacePhotoSection
	qBackPhotoSection
	qChestPhotoSection
	qOtherLocationPhotoSection
	qPrescriptionPreference
	qAcnePrevPrescriptionsSelect
	qAcnePrevPrescriptionsUsing
	qAcnePrevPrescriptionsEffective
	qAcnePrevPrescriptionsIrritate
	qAcnePrevPrescriptionsUsedMoreThanThreeMonths
	qAcnePrevPrescriptionsAnythingElse
	qAcnePrevOTCSelect
	qAcnePrevOTCUsing
	qAcnePrevOTCEffective
	qAcnePrevOTCIrritate
	qAcnePrevOTCTried
	qAcnePrevOTCAnythingElse
	qInsuranceCoverage
)

var (
	questionTags = map[string]questionTag{
		"q_onset_acne":                                   qAcneOnset,
		"q_skin_photo_comparison":                        qSkinPhotoComparison,
		"q_acne_worse":                                   qAcneWorse,
		"q_acne_worse_contributing_factors":              qAcneContributingFactors,
		"q_acne_symptoms":                                qAcneSymptoms,
		"q_acne_worse_period":                            qAcneWorsePeriod,
		"q_periods_regular":                              qPeriodsRegular,
		"q_skin_description":                             qSkinDescription,
		"q_acne_prev_treatment_types":                    qAcnePrevTreatmentTypes,
		"q_pregnancy_planning":                           qPregnancyPlanning,
		"q_current_medications":                          qCurrentMedications,
		"q_current_medications_entry":                    qCurrentMedicationsEntry,
		"q_length_current_medication":                    qLengthCurrentMedication,
		"q_allergic_medications":                         qAllergicMedications,
		"q_allergic_medication_entry":                    qAllergicMedicationEntry,
		"q_prev_skin_condition_diagnosis":                qPrevSkinConditionDiagnosis,
		"q_list_prev_skin_condition_diagnosis":           qListPrevSkinConditionDiagnosis,
		"q_other_conditions_acne":                        qOtherConditionsAcne,
		"q_face_photo_section":                           qFacePhotoSection,
		"q_back_photo_section":                           qBackPhotoSection,
		"q_chest_photo_section":                          qChestPhotoSection,
		"q_other_location_photo_section":                 qOtherLocationPhotoSection,
		"q_prescription_preference":                      qPrescriptionPreference,
		"q_acne_prev_prescriptions_select":               qAcnePrevPrescriptionsSelect,
		"q_using_prev_acne_prescription":                 qAcnePrevPrescriptionsUsing,
		"q_how_effective_prev_acne_prescription":         qAcnePrevPrescriptionsEffective,
		"q_use_more_three_months_prev_acne_prescription": qAcnePrevPrescriptionsUsedMoreThanThreeMonths,
		"q_irritate_skin_prev_acne_prescription":         qAcnePrevPrescriptionsIrritate,
		"q_anything_else_prev_acne_prescription":         qAcnePrevPrescriptionsAnythingElse,
		"q_acne_prev_otc_select":                         qAcnePrevOTCSelect,
		"q_acne_otc_product_tried":                       qAcnePrevOTCTried,
		"q_using_prev_acne_otc":                          qAcnePrevOTCUsing,
		"q_how_effective_prev_acne_otc":                  qAcnePrevOTCEffective,
		"q_irritate_skin_prev_acne_otc":                  qAcnePrevOTCIrritate,
		"q_anything_else_prev_acne_otc":                  qAcnePrevOTCAnythingElse,
		"q_insurance_coverage":                           qInsuranceCoverage,
	}
)

type potentialAnswerTag int

const (
	aSixToTwelveMonths potentialAnswerTag = iota
	aLessThanSixMonthsAgo
	aTwoOrMoreYearsAgo
	aAcneWorseYes
	aDiscoloration
	aScarring
	aPainfulToTouch
	aDeepLumps
	aAcneWorsePeriodNo
	aAcneWorsePeriodYes
	aPeriodsRegularYes
	aPeriodsRegularNo
	aSkinDescriptionNormal
	aSkinDescriptionOily
	aSkinDescriptionSensitive
	aSkinDescriptionOther
	aPrevTreatmentsTypeOTC
	aCurrentlyPregnant
	aNoPregnancyPlanning
	aCurrentMedicationsYes
	aCurrentMedicationsNo
	aTwoToFiveMonthsLength
	aLessThanOneMonthLength
	aSixToElevenMonthsLength
	aAllergicMedicationsYes
	aAllergicMedicationsNo
	aPrevSkinConditionDiagnosisYes
	aListPrevSkinConditionDiagnosisAcne
	aListPrevSkinConditionDiagnosisPsoriasis
	aListPrevSkinConditionDiagnosisEczema
	aNoneOfTheAboveOtherConditions
	aIntestinalInflammationOtherConditions
	aGenericRxOnly
	aPickedOrSqueezed
	aCreatedScars
	aBenzoylPeroxide
	aBenzaClin
	aCleanAndClear
	aAcneFree
	aNoxzema
	aClearasil
	aAcnePrevPrescriptionUsingYes
	aAcnePrevPrescriptionEffectiveSomewhat
	aAcnePrevPrescriptionUseMoreThanThreeMonthsNo
	aAcnePrevPrescriptionUseMoreThanThreeMonthsYes
	aAcnePrevPrescriptionIrritateSkinNo
	aAcnePrevPrescriptionIrritateSkinYes
	aProactiv
	aAcnePrevOTCUsingNo
	aAcnePrevOTCUsingYes
	aAcnePrevOTCEffectiveNo
	aAcnePrevOTCEffectiveSomewhat
	aAcnePrevOTCIrritateNo
	aAcnePrevOTCIrritateYes
	aPhotoComparisonMoreBlemishes
	aPhotoComparisonFewerBlemishes
	aPhotoComparisonAboutTheSame
	aAcneContributingFactorDiet
	aAcneContributingFactorSweating
	aAcneContributingFactorStress
	aAcneContributingFactorHormonalChanges
	aAcneContributingFactorNotSure
	aInsuranceCoverageGenericOnly
	aInsuranceCoverageIDK
	aInsuranceCoverageNoInsurance
)

var (
	answerTags = map[string]potentialAnswerTag{
		"a_six_twelve_months_ago":                            aSixToTwelveMonths,
		"a_less_six_months":                                  aLessThanSixMonthsAgo,
		"a_twa_plus_years":                                   aTwoOrMoreYearsAgo,
		"a_yes_acne_worse":                                   aAcneWorseYes,
		"a_acne_worse_no":                                    aAcneWorsePeriodNo,
		"a_acne_worse_yes":                                   aAcneWorsePeriodYes,
		"a_periods_regular_yes":                              aPeriodsRegularYes,
		"a_periods_regular_no":                               aPeriodsRegularNo,
		"a_oil_skin":                                         aSkinDescriptionOily,
		"a_normal_skin":                                      aSkinDescriptionNormal,
		"a_sensitive_skin":                                   aSkinDescriptionSensitive,
		"a_otc_prev_treatment_type":                          aPrevTreatmentsTypeOTC,
		"a_yes_pregnancy_planning":                           aCurrentlyPregnant,
		"a_na_pregnancy_planning":                            aNoPregnancyPlanning,
		"a_current_medications_yes":                          aCurrentMedicationsYes,
		"a_current_medications_no":                           aCurrentMedicationsNo,
		"a_length_current_medication_two_five_months":        aTwoToFiveMonthsLength,
		"a_length_current_medication_less_than_month":        aLessThanOneMonthLength,
		"a_length_current_medication_six_eleven_months":      aSixToElevenMonthsLength,
		"a_yes_allergic_medications":                         aAllergicMedicationsYes,
		"a_na_allergic_medications":                          aAllergicMedicationsNo,
		"a_yes_prev_skin_diagnosis":                          aPrevSkinConditionDiagnosisYes,
		"a_acne_skin_diagnosis":                              aListPrevSkinConditionDiagnosisAcne,
		"a_psoriasis_skin_diagnosis":                         aListPrevSkinConditionDiagnosisPsoriasis,
		"a_eczema_skin_diagnosis":                            aListPrevSkinConditionDiagnosisEczema,
		"a_other_condition_acne_none":                        aNoneOfTheAboveOtherConditions,
		"a_other_condition_acne_intestinal_inflammation":     aIntestinalInflammationOtherConditions,
		"a_generic_only":                                     aGenericRxOnly,
		"a_picked_or_squeezed":                               aPickedOrSqueezed,
		"a_deep_lumps":                                       aDeepLumps,
		"a_created_scars":                                    aCreatedScars,
		"a_discoloration":                                    aDiscoloration,
		"a_painful_touch":                                    aPainfulToTouch,
		"a_benzaclin":                                        aBenzaClin,
		"a_benzoyl_peroxide":                                 aBenzoylPeroxide,
		"a_acne_free":                                        aAcneFree,
		"a_clean_clear":                                      aCleanAndClear,
		"a_clearasil":                                        aClearasil,
		"a_noxzema":                                          aNoxzema,
		"a_using_prev_prescription_yes":                      aAcnePrevPrescriptionUsingYes,
		"a_how_effective_prev_acne_prescription_somewhat":    aAcnePrevPrescriptionEffectiveSomewhat,
		"a_use_more_three_months_prev_acne_prescription_no":  aAcnePrevPrescriptionUseMoreThanThreeMonthsNo,
		"a_use_more_three_months_prev_acne_prescription_yes": aAcnePrevPrescriptionUseMoreThanThreeMonthsYes,
		"a_irritate_skin_prev_acne_prescription_no":          aAcnePrevPrescriptionIrritateSkinNo,
		"a_irritate_skin_prev_acne_prescription_yes":         aAcnePrevPrescriptionIrritateSkinYes,
		"a_proactiv":                                         aProactiv,
		"a_using_prev_otc_no":                                aAcnePrevOTCUsingNo,
		"a_using_prev_otc_yes":                               aAcnePrevOTCUsingYes,
		"a_how_effective_prev_acne_otc_not":                  aAcnePrevOTCEffectiveNo,
		"a_how_effective_prev_acne_otc_somewhat":             aAcnePrevOTCEffectiveSomewhat,
		"a_irritate_skin_prev_acne_otc_no":                   aAcnePrevOTCIrritateNo,
		"a_irritate_skin_prev_acne_otc_yes":                  aAcnePrevOTCIrritateYes,
		"a_more_acne_blemishes_photo_comparison":             aPhotoComparisonMoreBlemishes,
		"a_fewer_acne_blemishes_photo_comparison":            aPhotoComparisonFewerBlemishes,
		"a_about_the_same_photo_comparison":                  aPhotoComparisonAboutTheSame,
		"a_acne_worse_diet":                                  aAcneContributingFactorDiet,
		"a_acne_worse_sweating_and_sports":                   aAcneContributingFactorSweating,
		"a_acne_worse_hormonal_changes":                      aAcneContributingFactorHormonalChanges,
		"a_acne_worse_stress":                                aAcneContributingFactorStress,
		"a_acne_worse_none_or_not_sure":                      aAcneContributingFactorNotSure,
		"a_insurance_generic_only":                           aInsuranceCoverageGenericOnly,
		"a_insurance_idk":                                    aInsuranceCoverageIDK,
		"a_no_insurance":                                     aInsuranceCoverageNoInsurance,
		"a_other_skin":                                       aSkinDescriptionOther,
	}
)

type photoSlotType int

const (
	photoSlotFaceFront photoSlotType = iota
	photoSlotFaceRight
	photoSlotFaceLeft
	photoSlotOther
	photoSlotBack
	photoSlotChest
)

var (
	photoSlotTypes = map[string]photoSlotType{
		"photo_slot_face_right": photoSlotFaceRight,
		"photo_slot_face_left":  photoSlotFaceLeft,
		"photo_slot_face_front": photoSlotFaceFront,
		"photo_slot_other":      photoSlotOther,
		"photo_slot_chest":      photoSlotChest,
		"photo_slot_back":       photoSlotBack,
	}
)

type answerTemplate struct {
	AnswerText         string
	AnswerTag          potentialAnswerTag
	SubquestionAnswers map[questionTag][]*answerTemplate
}

type photoSlotTemplate struct {
	PhotoSlotTag photoSlotType
	PhotoURL     string
	Name         string
}

type photoSectionTemplate struct {
	SectionName string
	QuestionTag questionTag
	PhotoSlots  []*photoSlotTemplate
}

type trainingCaseTemplate struct {
	Name                  string
	PatientToCreate       *common.Patient
	IntakeToSubmit        map[questionTag][]*answerTemplate
	PhotoSectionsToSubmit []*photoSectionTemplate
	VisitMessage          string
}
