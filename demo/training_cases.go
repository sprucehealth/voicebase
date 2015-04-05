package demo

import (
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/pharmacy"
)

var trainingCases = []*trainingCaseTemplate{
	trainingCase1,
	trainingCase2,
	trainingCase3,
	trainingCase4,
	trainingCase5,
}

var trainingCase1 = &trainingCaseTemplate{
	Name: "tc1",
	PatientToCreate: &common.Patient{
		FirstName: "Rachel",
		LastName:  "Green",
		Gender:    "female",
		DOB: encoding.Date{
			Year:  1988,
			Month: 9,
			Day:   27,
		},
		ZipCode: "94401",
		PhoneNumbers: []*common.PhoneNumber{&common.PhoneNumber{
			Phone: "6505552656",
			Type:  "Cell",
		},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceID:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PharmacySourceSurescripts,
		},
		PatientAddress: &common.Address{
			AddressLine1: "111 Lincoln Street",
			AddressLine2: "Apt 1112",
			City:         "San Mateo",
			State:        "California",
			ZipCode:      "94401",
		},
	},
	PhotoSectionsToSubmit: []*photoSectionTemplate{
		&photoSectionTemplate{
			SectionName: "Face",
			QuestionTag: qFacePhotoSection,
			PhotoSlots: []*photoSlotTemplate{
				{
					Name:         "Front",
					PhotoSlotTag: photoSlotFaceFront,
					PhotoURL:     "tc1_face1.jpg",
				},
				{
					Name:         "Left Side",
					PhotoSlotTag: photoSlotFaceLeft,
					PhotoURL:     "tc1_face2.jpg",
				},
			},
		},
	},
	IntakeToSubmit: map[questionTag][]*answerTemplate{
		qSkinDescription: []*answerTemplate{
			{AnswerTag: aSkinDescriptionSensitive},
			{AnswerTag: aSkinDescriptionOther},
			{AnswerText: "Combination"},
		},
		qAcneSymptoms: []*answerTemplate{
			{AnswerTag: aPainfulToTouch},
			{AnswerTag: aDiscoloration},
		},
		qAcneWorse: []*answerTemplate{
			{AnswerTag: aAcneWorseYes},
		},
		qAcneContributingFactors: []*answerTemplate{
			{AnswerTag: aAcneContributingFactorNotSure},
		},
		qAcneWorsePeriod: []*answerTemplate{
			{AnswerTag: aAcneWorsePeriodNo},
		},
		qAcnePrevPrescriptionsSelect: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aBenzaClin,
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qAcnePrevPrescriptionsUsing: []*answerTemplate{
							{AnswerTag: aAcnePrevPrescriptionUsingYes},
						},
					},
					{
						qAcnePrevPrescriptionsEffective: []*answerTemplate{
							{AnswerTag: aAcnePrevPrescriptionEffectiveSomewhat},
						},
					},
					{
						qAcnePrevPrescriptionsUsedMoreThanThreeMonths: []*answerTemplate{
							{AnswerTag: aAcnePrevPrescriptionUseMoreThanThreeMonthsYes},
						},
					},
					{
						qAcnePrevPrescriptionsIrritate: []*answerTemplate{
							{AnswerTag: aAcnePrevPrescriptionIrritateSkinNo},
						},
					},
				},
			},
		},
		qAcnePrevOTCSelect: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aCleanAndClear,
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qAcnePrevOTCUsing: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCUsingYes},
						},
					},
					{
						qAcnePrevOTCEffective: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCEffectiveSomewhat},
						},
					},
					{
						qAcnePrevOTCIrritate: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCIrritateNo},
						},
					},
				},
			},
		},
		qAcneOnset: []*answerTemplate{
			{AnswerTag: aTwoOrMoreYearsAgo},
		},
		qSkinPhotoComparison: []*answerTemplate{
			{AnswerTag: aPhotoComparisonAboutTheSame},
		},
		qAllergicMedications: []*answerTemplate{
			{AnswerTag: aAllergicMedicationsNo},
		},
		qCurrentMedications: []*answerTemplate{
			{AnswerTag: aCurrentMedicationsNo},
		},
		qInsuranceCoverage: []*answerTemplate{
			{AnswerTag: aInsuranceCoverageGenericOnly},
		},
		qPrevSkinConditionDiagnosis: []*answerTemplate{
			{AnswerTag: aPrevSkinConditionDiagnosisYes},
		},
		qListPrevSkinConditionDiagnosis: []*answerTemplate{
			{AnswerTag: aListPrevSkinConditionDiagnosisEczema},
		},
		qOtherConditionsAcne: []*answerTemplate{
			{AnswerTag: aNoneOfTheAboveOtherConditions},
		},
		qPregnancyPlanning: []*answerTemplate{
			{AnswerTag: aCurrentlyPregnant},
		},
	},
	VisitMessage: "I want to make sure that any medications prescribed will be safe to take while pregnant and breast feeding.  Is there anything that you can prescribe that would be safe for my baby?",
}

var trainingCase2 = &trainingCaseTemplate{
	Name: "tc2",
	PatientToCreate: &common.Patient{
		FirstName: "Donald",
		LastName:  "Parson",
		Gender:    "male",
		DOB: encoding.Date{
			Year:  1982,
			Month: 4,
			Day:   5,
		},
		ZipCode: "94401",
		PhoneNumbers: []*common.PhoneNumber{&common.PhoneNumber{
			Phone: "4155236507",
			Type:  "Cell",
		},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceID:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PharmacySourceSurescripts,
		},
		PatientAddress: &common.Address{
			AddressLine1: "111 Lincoln Street",
			AddressLine2: "Apt 1112",
			City:         "San Mateo",
			State:        "California",
			ZipCode:      "94401",
		},
	},
	PhotoSectionsToSubmit: []*photoSectionTemplate{
		&photoSectionTemplate{
			SectionName: "Face",
			QuestionTag: qFacePhotoSection,
			PhotoSlots: []*photoSlotTemplate{
				{
					Name:         "Front",
					PhotoSlotTag: photoSlotFaceFront,
					PhotoURL:     "tc2_face1.jpg",
				},
				{
					Name:         "Left Side",
					PhotoSlotTag: photoSlotFaceLeft,
					PhotoURL:     "tc2_face2.jpg",
				},
			},
		},
	},
	IntakeToSubmit: map[questionTag][]*answerTemplate{
		qSkinDescription: []*answerTemplate{
			{AnswerTag: aSkinDescriptionNormal},
		},
		qAcneSymptoms: []*answerTemplate{
			{AnswerTag: aDiscoloration},
			{AnswerTag: aCreatedScars},
		},
		qAcneWorse: []*answerTemplate{
			{AnswerTag: aAcneWorseYes},
		},
		qAcneContributingFactors: []*answerTemplate{
			{AnswerTag: aAcneContributingFactorDiet},
		},
		qAcnePrevOTCSelect: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAcneFree,
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qAcnePrevOTCUsing: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCUsingNo},
						},
					},
					{
						qAcnePrevOTCEffective: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCEffectiveNo},
						},
					},
					{
						qAcnePrevOTCIrritate: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCIrritateNo},
						},
					},
				},
			},
			&answerTemplate{
				AnswerTag: aClearasil,
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qAcnePrevOTCUsing: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCUsingYes},
						},
					},
					{
						qAcnePrevOTCEffective: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCEffectiveSomewhat},
						},
					},
					{
						qAcnePrevOTCIrritate: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCIrritateNo},
						},
					},
				},
			},
			&answerTemplate{
				AnswerTag: aProactiv,
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qAcnePrevOTCUsing: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCUsingNo},
						},
					},
					{
						qAcnePrevOTCEffective: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCEffectiveNo},
						},
					},
					{
						qAcnePrevOTCIrritate: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCIrritateYes},
						},
					},
				},
			},
		},
		qAcneOnset: []*answerTemplate{
			{AnswerTag: aLessThanSixMonthsAgo},
		},
		qSkinPhotoComparison: []*answerTemplate{
			{AnswerTag: aPhotoComparisonMoreBlemishes},
		},
		qAllergicMedications: []*answerTemplate{
			{AnswerTag: aAllergicMedicationsNo},
		},
		qCurrentMedications: []*answerTemplate{
			{AnswerTag: aCurrentMedicationsYes},
		},
		qCurrentMedicationsEntry: []*answerTemplate{
			&answerTemplate{
				AnswerText: "Advair Diskus",
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qLengthCurrentMedication: []*answerTemplate{
							{AnswerTag: aLessThanOneMonthLength},
						},
					},
				},
			},
		},
		qInsuranceCoverage: []*answerTemplate{
			{AnswerTag: aInsuranceCoverageIDK},
		},
		qPrevSkinConditionDiagnosis: []*answerTemplate{
			{AnswerTag: aPrevSkinConditionDiagnosisYes},
		},
		qListPrevSkinConditionDiagnosis: []*answerTemplate{
			{AnswerTag: aListPrevSkinConditionDiagnosisPsoriasis},
		},
		qOtherConditionsAcne: []*answerTemplate{
			{AnswerTag: aNoneOfTheAboveOtherConditions},
		},
	},
	VisitMessage: "How will any prescribed medication react with my psoriasis?",
}

var trainingCase3 = &trainingCaseTemplate{
	Name: "tc3",
	PatientToCreate: &common.Patient{
		FirstName: "Hope",
		LastName:  "Alejandro",
		Gender:    "female",
		DOB: encoding.Date{
			Year:  1980,
			Month: 6,
			Day:   1,
		},
		ZipCode: "94020",
		PhoneNumbers: []*common.PhoneNumber{
			{
				Phone: "2068773590",
				Type:  "Cell",
			},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceID:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PharmacySourceSurescripts,
		},
		PatientAddress: &common.Address{
			AddressLine1: "111 Lincoln Street",
			AddressLine2: "Apt 1112",
			City:         "San Mateo",
			State:        "California",
			ZipCode:      "94401",
		},
	},
	PhotoSectionsToSubmit: []*photoSectionTemplate{
		&photoSectionTemplate{
			SectionName: "Face",
			QuestionTag: qFacePhotoSection,
			PhotoSlots: []*photoSlotTemplate{
				{
					Name:         "Front",
					PhotoSlotTag: photoSlotFaceFront,
					PhotoURL:     "tc3_face1.jpg",
				},
				{
					Name:         "Left",
					PhotoSlotTag: photoSlotFaceLeft,
					PhotoURL:     "tc3_face2.jpg",
				},
			},
		},
	},
	IntakeToSubmit: map[questionTag][]*answerTemplate{
		qSkinDescription: []*answerTemplate{
			{AnswerTag: aSkinDescriptionNormal},
			{AnswerTag: aSkinDescriptionOily},
			{AnswerTag: aSkinDescriptionSensitive},
		},
		qAcneSymptoms: []*answerTemplate{
			{AnswerTag: aPickedOrSqueezed},
			{AnswerTag: aDeepLumps},
		},
		qAcneWorse: []*answerTemplate{
			{AnswerTag: aAcneWorseYes},
		},
		qAcneWorsePeriod: []*answerTemplate{
			{AnswerTag: aAcneWorsePeriodYes},
		},
		qPeriodsRegular: []*answerTemplate{
			{AnswerTag: aPeriodsRegularYes},
		},
		qAcneContributingFactors: []*answerTemplate{
			{AnswerTag: aAcneContributingFactorHormonalChanges},
		},
		qAcnePrevPrescriptionsSelect: []*answerTemplate{
			&answerTemplate{
				AnswerText: "Oral Birth Control",
				AnswerTag:  aAcnePrevPrescriptionOther,
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qAcnePrevPrescriptionsUsing: []*answerTemplate{
							{AnswerTag: aAcnePrevPrescriptionUsingYes},
						},
					},
					{
						qAcnePrevPrescriptionsEffective: []*answerTemplate{
							{AnswerTag: aAcnePrevPrescriptionEffectiveSomewhat},
						},
					},
					{
						qAcnePrevPrescriptionsUsedMoreThanThreeMonths: []*answerTemplate{
							{AnswerTag: aAcnePrevPrescriptionUseMoreThanThreeMonthsYes},
						},
					},
					{
						qAcnePrevPrescriptionsIrritate: []*answerTemplate{
							{AnswerTag: aAcnePrevPrescriptionIrritateSkinNo},
						},
					},
				},
			},
		},
		qAcnePrevOTCSelect: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aCleanAndClear,
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qAcnePrevOTCUsing: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCUsingNo},
						},
					},
					{
						qAcnePrevOTCEffective: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCEffectiveSomewhat},
						},
					},
					{
						qAcnePrevOTCIrritate: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCIrritateNo},
						},
					},
				},
			},
			&answerTemplate{
				AnswerTag: aNoxzema,
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qAcnePrevOTCUsing: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCUsingYes},
						},
					},
					{
						qAcnePrevOTCEffective: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCEffectiveSomewhat},
						},
					},
					{
						qAcnePrevOTCIrritate: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCIrritateNo},
						},
					},
				},
			},
		},
		qAcneOnset: []*answerTemplate{
			{AnswerTag: aSixToTwelveMonths},
		},
		qSkinPhotoComparison: []*answerTemplate{
			{AnswerTag: aPhotoComparisonFewerBlemishes},
		},
		qAllergicMedications: []*answerTemplate{
			{AnswerTag: aAllergicMedicationsNo},
		},
		qCurrentMedications: []*answerTemplate{
			{AnswerTag: aCurrentMedicationsYes},
		},
		qPregnancyPlanning: []*answerTemplate{
			{AnswerTag: aNoPregnancyPlanning},
		},
		qCurrentMedicationsEntry: []*answerTemplate{
			&answerTemplate{
				AnswerText: "Yasmin",
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qLengthCurrentMedication: []*answerTemplate{
							{AnswerTag: aSixToElevenMonthsLength},
						},
					},
				},
			},
		},
		qInsuranceCoverage: []*answerTemplate{
			{AnswerTag: aInsuranceCoverageNoInsurance},
		},
		qPrevSkinConditionDiagnosis: []*answerTemplate{
			{AnswerTag: aPrevSkinConditionDiagnosisYes},
		},
		qListPrevSkinConditionDiagnosis: []*answerTemplate{
			{AnswerTag: aListPrevSkinConditionDiagnosisAcne},
		},
		qOtherConditionsAcne: []*answerTemplate{
			{AnswerTag: aIntestinalInflammationOtherConditions},
		},
	},
	VisitMessage: "Will my current condition and current medication be considered when prescribing new medications?",
}

var trainingCase4 = &trainingCaseTemplate{
	Name: "tc4",
	PatientToCreate: &common.Patient{
		FirstName: "Ralph",
		LastName:  "Flower",
		Gender:    "male",
		DOB: encoding.Date{
			Year:  1987,
			Month: 4,
			Day:   1,
		},
		ZipCode: "94020",
		PhoneNumbers: []*common.PhoneNumber{&common.PhoneNumber{
			Phone: "6504933158",
			Type:  "Cell",
		},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceID:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PharmacySourceSurescripts,
		},
		PatientAddress: &common.Address{
			AddressLine1: "111 Lincoln Street",
			AddressLine2: "Apt 1112",
			City:         "San Mateo",
			State:        "California",
			ZipCode:      "94401",
		},
	},
	PhotoSectionsToSubmit: []*photoSectionTemplate{
		&photoSectionTemplate{
			SectionName: "Chest",
			QuestionTag: qChestPhotoSection,
			PhotoSlots: []*photoSlotTemplate{
				{
					Name:         "Chest",
					PhotoSlotTag: photoSlotChest,
					PhotoURL:     "tc4_chest.jpg",
				},
			},
		},
		&photoSectionTemplate{
			SectionName: "Back",
			QuestionTag: qBackPhotoSection,
			PhotoSlots: []*photoSlotTemplate{
				{
					Name:         "Back",
					PhotoSlotTag: photoSlotBack,
					PhotoURL:     "tc4_back.jpg",
				},
			},
		},
	},
	IntakeToSubmit: map[questionTag][]*answerTemplate{
		qSkinDescription: []*answerTemplate{
			{AnswerTag: aSkinDescriptionNormal},
			{AnswerTag: aSkinDescriptionSensitive},
		},
		qAcneSymptoms: []*answerTemplate{
			{AnswerTag: aPickedOrSqueezed},
			{AnswerTag: aDeepLumps},
			{AnswerTag: aPainfulToTouch},
			{AnswerTag: aDiscoloration},
			{AnswerTag: aCreatedScars},
		},
		qAcneWorse: []*answerTemplate{
			{AnswerTag: aAcneWorseYes},
		},
		qAcneContributingFactors: []*answerTemplate{
			{AnswerTag: aAcneContributingFactorSweating},
		},
		qAcnePrevOTCSelect: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aCleanAndClear,
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qAcnePrevOTCUsing: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCUsingNo},
						},
					},
					{
						qAcnePrevOTCEffective: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCEffectiveNo},
						},
					},
					{
						qAcnePrevOTCIrritate: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCIrritateNo},
						},
					},
				},
			},
			&answerTemplate{
				AnswerTag: aNeutrogena,
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qAcnePrevOTCUsing: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCUsingYes},
						},
					},
					{
						qAcnePrevOTCEffective: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCEffectiveSomewhat},
						},
					},
					{
						qAcnePrevOTCIrritate: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCIrritateNo},
						},
					},
				},
			},
		},
		qAcneOnset: []*answerTemplate{
			{AnswerTag: aLessThanSixMonthsAgo},
		},
		qSkinPhotoComparison: []*answerTemplate{
			{AnswerTag: aPhotoComparisonFewerBlemishes},
		},
		qAllergicMedications: []*answerTemplate{
			{AnswerTag: aAllergicMedicationsNo},
		},
		qCurrentMedications: []*answerTemplate{
			{AnswerTag: aCurrentMedicationsNo},
		},
		qInsuranceCoverage: []*answerTemplate{
			{AnswerTag: aInsuranceCoverageBrandAndGeneric},
		},
		qPrevSkinConditionDiagnosis: []*answerTemplate{
			{AnswerTag: aPrevSkinConditionDiagnosisYes},
		},
		qListPrevSkinConditionDiagnosis: []*answerTemplate{
			{AnswerTag: aListPrevSkinConditionDiagnosisRosacea},
		},
		qOtherConditionsAcne: []*answerTemplate{
			{AnswerTag: aNoneOfTheAboveOtherConditions},
		},
	},
	VisitMessage: "I've been diagnosed with Rosacea in the past and still have the symptoms.  Would you be able to prescribe something that will either help my Rosacea or that won't make my symptoms worse?",
}

var trainingCase5 = &trainingCaseTemplate{
	Name: "tc5",
	PatientToCreate: &common.Patient{
		FirstName: "Willie",
		LastName:  "Todd",
		Gender:    "female",
		DOB: encoding.Date{
			Year:  1990,
			Month: 2,
			Day:   21,
		},
		ZipCode: "94105",
		PhoneNumbers: []*common.PhoneNumber{&common.PhoneNumber{
			Phone: "2068773590",
			Type:  "Cell",
		},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceID:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PharmacySourceSurescripts,
		},
		PatientAddress: &common.Address{
			AddressLine1: "111 Lincoln Street",
			AddressLine2: "Apt 1112",
			City:         "San Mateo",
			State:        "California",
			ZipCode:      "94401",
		},
	},
	PhotoSectionsToSubmit: []*photoSectionTemplate{
		&photoSectionTemplate{
			SectionName: "Face",
			QuestionTag: qOtherLocationPhotoSection,
			PhotoSlots: []*photoSlotTemplate{
				{
					Name:         "Eye",
					PhotoSlotTag: photoSlotOther,
					PhotoURL:     "tc5_eye.jpg",
				},
				{
					Name:         "Forehead",
					PhotoSlotTag: photoSlotFaceFront,
					PhotoURL:     "tc5_forehead.jpg",
				},
			},
		},
	},
	IntakeToSubmit: map[questionTag][]*answerTemplate{
		qSkinDescription: []*answerTemplate{
			{AnswerTag: aSkinDescriptionDry},
		},
		qAcneSymptoms: []*answerTemplate{
			{AnswerTag: aPainfulToTouch},
			{AnswerTag: aDeepLumps},
		},
		qAcneWorse: []*answerTemplate{
			{AnswerTag: aAcneWorseNo},
		},
		qAcneWorsePeriod: []*answerTemplate{
			{AnswerTag: aAcneWorsePeriodNo},
		},
		qAcnePrevOTCSelect: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aPanOxyl,
				SubquestionAnswers: []map[questionTag][]*answerTemplate{
					{
						qAcnePrevOTCUsing: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCUsingYes},
						},
					},
					{
						qAcnePrevOTCEffective: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCEffectiveNo},
						},
					},
					{
						qAcnePrevOTCIrritate: []*answerTemplate{
							{AnswerTag: aAcnePrevOTCIrritateNo},
						},
					},
				},
			},
		},
		qAcneOnset: []*answerTemplate{
			{AnswerTag: aSixToTwelveMonths},
		},
		qSkinPhotoComparison: []*answerTemplate{
			{AnswerTag: aPhotoComparisonAboutTheSame},
		},
		qAllergicMedications: []*answerTemplate{
			{AnswerTag: aAllergicMedicationsNo},
		},
		qCurrentMedications: []*answerTemplate{
			{AnswerTag: aCurrentMedicationsNo},
		},
		qPregnancyPlanning: []*answerTemplate{
			{AnswerTag: aNoPregnancyPlanning},
		},
		qInsuranceCoverage: []*answerTemplate{
			{AnswerTag: aInsuranceCoverageIDK},
		},
		qPrevSkinConditionDiagnosis: []*answerTemplate{
			{AnswerTag: aPrevSkinConditionDiagnosisNo},
		},
		qOtherConditionsAcne: []*answerTemplate{
			{AnswerTag: aPolycysticOvarySyndrome},
		},
	},
	VisitMessage: "I don't know if my medical insurance covers brand name drugs.  Would you make sure to please prescribe a generic version?",
}
