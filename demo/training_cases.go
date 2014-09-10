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
}

var trainingCase1 = &trainingCaseTemplate{
	Name: "training_case_1",
	PatientToCreate: &common.Patient{
		FirstName: "Training",
		LastName:  "Patient 1",
		Gender:    "female",
		DOB: encoding.DOB{
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
			SourceId:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
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
				&photoSlotTemplate{
					Name:         "Front",
					PhotoSlotTag: photoSlotFaceFront,
					PhotoURL:     "tc1_face1.jpg",
				},
				&photoSlotTemplate{
					Name:         "Left Side",
					PhotoSlotTag: photoSlotFaceLeft,
					PhotoURL:     "tc1_face2.jpg",
				},
			},
		},
	},
	IntakeToSubmit: map[questionTag][]*answerTemplate{
		qSkinDescription: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aSkinDescriptionSensitive,
			},
			&answerTemplate{
				AnswerTag: aSkinDescriptionOther,
			},
			&answerTemplate{
				AnswerText: "Combination",
			},
		},
		qAcneSymptoms: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aPainfulToTouch,
			},
			&answerTemplate{
				AnswerTag: aDiscoloration,
			},
		},
		qAcneWorse: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAcneWorseYes,
			},
		},
		qAcneContributingFactors: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAcneContributingFactorNotSure,
			},
		},
		qAcneWorsePeriod: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAcneWorsePeriodNo,
			},
		},
		qAcnePrevPrescriptionsSelect: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aBenzaClin,
				SubquestionAnswers: map[questionTag][]*answerTemplate{
					qAcnePrevPrescriptionsUsing: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevPrescriptionUsingYes,
						},
					},
					qAcnePrevPrescriptionsEffective: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevPrescriptionEffectiveSomewhat,
						},
					},
					qAcnePrevPrescriptionsUsedMoreThanThreeMonths: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevPrescriptionUseMoreThanThreeMonthsYes,
						},
					},
					qAcnePrevPrescriptionsIrritate: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevPrescriptionIrritateSkinNo,
						},
					},
				},
			},
		},
		qAcnePrevOTCSelect: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aCleanAndClear,
				SubquestionAnswers: map[questionTag][]*answerTemplate{
					qAcnePrevOTCUsing: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCUsingYes,
						},
					},
					qAcnePrevOTCEffective: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCEffectiveSomewhat,
						},
					},
					qAcnePrevOTCIrritate: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCIrritateNo,
						},
					},
				},
			},
		},
		qAcneOnset: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aTwoOrMoreYearsAgo,
			},
		},
		qSkinPhotoComparison: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aPhotoComparisonAboutTheSame,
			},
		},
		qAllergicMedications: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAllergicMedicationsNo,
			},
		},
		qCurrentMedications: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aCurrentMedicationsNo,
			},
		},
		qInsuranceCoverage: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aInsuranceCoverageGenericOnly,
			},
		},
		qPrevSkinConditionDiagnosis: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aPrevSkinConditionDiagnosisYes,
			},
		},
		qListPrevSkinConditionDiagnosis: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aListPrevSkinConditionDiagnosisEczema,
			},
		},
		qOtherConditionsAcne: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aNoneOfTheAboveOtherConditions,
			},
		},
		qPregnancyPlanning: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aCurrentlyPregnant,
			},
		},
	},
}

var trainingCase2 = &trainingCaseTemplate{
	Name: "training_case_2",
	PatientToCreate: &common.Patient{
		FirstName: "Training",
		LastName:  "Patient 2",
		Gender:    "male",
		DOB: encoding.DOB{
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
			SourceId:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
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
				&photoSlotTemplate{
					Name:         "Front",
					PhotoSlotTag: photoSlotFaceFront,
					PhotoURL:     "tc2_face1.jpg",
				},
				&photoSlotTemplate{
					Name:         "Left Side",
					PhotoSlotTag: photoSlotFaceLeft,
					PhotoURL:     "tc2_face2.jpg",
				},
			},
		},
	},
	IntakeToSubmit: map[questionTag][]*answerTemplate{
		qSkinDescription: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aSkinDescriptionNormal,
			},
		},
		qAcneSymptoms: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aDiscoloration,
			},
			&answerTemplate{
				AnswerTag: aCreatedScars,
			},
		},
		qAcneWorse: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAcneWorseYes,
			},
		},
		qAcneContributingFactors: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAcneContributingFactorDiet,
			},
		},
		qAcnePrevOTCSelect: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAcneFree,
				SubquestionAnswers: map[questionTag][]*answerTemplate{
					qAcnePrevOTCUsing: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCUsingNo,
						},
					},
					qAcnePrevOTCEffective: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCEffectiveNo,
						},
					},
					qAcnePrevOTCIrritate: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCIrritateNo,
						},
					},
				},
			},
			&answerTemplate{
				AnswerTag: aClearasil,
				SubquestionAnswers: map[questionTag][]*answerTemplate{
					qAcnePrevOTCUsing: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCUsingYes,
						},
					},
					qAcnePrevOTCEffective: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCEffectiveSomewhat,
						},
					},
					qAcnePrevOTCIrritate: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCIrritateNo,
						},
					},
				},
			},
			&answerTemplate{
				AnswerTag: aProactiv,
				SubquestionAnswers: map[questionTag][]*answerTemplate{
					qAcnePrevOTCUsing: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCUsingNo,
						},
					},
					qAcnePrevOTCEffective: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCEffectiveNo,
						},
					},
					qAcnePrevOTCIrritate: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCIrritateYes,
						},
					},
				},
			},
		},
		qAcneOnset: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aLessThanSixMonthsAgo,
			},
		},
		qSkinPhotoComparison: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aPhotoComparisonMoreBlemishes,
			},
		},
		qAllergicMedications: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAllergicMedicationsNo,
			},
		},
		qCurrentMedications: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aCurrentMedicationsYes,
			},
		},
		qCurrentMedicationsEntry: []*answerTemplate{
			&answerTemplate{
				AnswerText: "Advair Diskus",
				SubquestionAnswers: map[questionTag][]*answerTemplate{
					qLengthCurrentMedication: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aLessThanOneMonthLength,
						},
					},
				},
			},
		},
		qInsuranceCoverage: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aInsuranceCoverageIDK,
			},
		},
		qPrevSkinConditionDiagnosis: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aPrevSkinConditionDiagnosisYes,
			},
		},
		qListPrevSkinConditionDiagnosis: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aListPrevSkinConditionDiagnosisPsoriasis,
			},
		},
		qOtherConditionsAcne: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aNoneOfTheAboveOtherConditions,
			},
		},
	},
	VisitMessage: "How will any prescribed medication react with my psoriasis?",
}

var trainingCase3 = &trainingCaseTemplate{
	Name: "training_case_3",
	PatientToCreate: &common.Patient{
		FirstName: "Training",
		LastName:  "Patient 3",
		Gender:    "female",
		DOB: encoding.DOB{
			Year:  1980,
			Month: 6,
			Day:   1,
		},
		ZipCode: "94020",
		PhoneNumbers: []*common.PhoneNumber{&common.PhoneNumber{
			Phone: "2068773590",
			Type:  "Cell",
		},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceId:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
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
				&photoSlotTemplate{
					Name:         "Front",
					PhotoSlotTag: photoSlotFaceFront,
					PhotoURL:     "tc3_face1.jpg",
				},
				&photoSlotTemplate{
					Name:         "Left",
					PhotoSlotTag: photoSlotFaceLeft,
					PhotoURL:     "tc3_face2.jpg",
				},
			},
		},
	},
	IntakeToSubmit: map[questionTag][]*answerTemplate{
		qSkinDescription: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aSkinDescriptionNormal,
			},
			&answerTemplate{
				AnswerTag: aSkinDescriptionOily,
			},
			&answerTemplate{
				AnswerTag: aSkinDescriptionSensitive,
			},
		},
		qAcneSymptoms: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aPickedOrSqueezed,
			},
			&answerTemplate{
				AnswerTag: aDeepLumps,
			},
		},
		qAcneWorse: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAcneWorseYes,
			},
		},
		qAcneWorsePeriod: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAcneWorsePeriodYes,
			},
		},
		qPeriodsRegular: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aPeriodsRegularYes,
			},
		},
		qAcneContributingFactors: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAcneContributingFactorHormonalChanges,
			},
		},
		qAcnePrevPrescriptionsSelect: []*answerTemplate{
			&answerTemplate{
				AnswerText: "Oral Birth Control",
				SubquestionAnswers: map[questionTag][]*answerTemplate{
					qAcnePrevPrescriptionsUsing: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevPrescriptionUsingYes,
						},
					},
					qAcnePrevPrescriptionsEffective: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevPrescriptionEffectiveSomewhat,
						},
					},
					qAcnePrevPrescriptionsUsedMoreThanThreeMonths: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevPrescriptionUseMoreThanThreeMonthsYes,
						},
					},
					qAcnePrevPrescriptionsIrritate: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevPrescriptionIrritateSkinNo,
						},
					},
				},
			},
		},
		qAcnePrevOTCSelect: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aCleanAndClear,
				SubquestionAnswers: map[questionTag][]*answerTemplate{
					qAcnePrevOTCUsing: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCUsingNo,
						},
					},
					qAcnePrevOTCEffective: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCEffectiveSomewhat,
						},
					},
					qAcnePrevOTCIrritate: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCIrritateNo,
						},
					},
				},
			},
			&answerTemplate{
				AnswerTag: aNoxzema,
				SubquestionAnswers: map[questionTag][]*answerTemplate{
					qAcnePrevOTCUsing: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCUsingYes,
						},
					},
					qAcnePrevOTCEffective: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCEffectiveSomewhat,
						},
					},
					qAcnePrevOTCIrritate: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aAcnePrevOTCIrritateNo,
						},
					},
				},
			},
		},
		qAcneOnset: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aSixToTwelveMonths,
			},
		},
		qSkinPhotoComparison: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aPhotoComparisonFewerBlemishes,
			},
		},
		qAllergicMedications: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aAllergicMedicationsNo,
			},
		},
		qCurrentMedications: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aCurrentMedicationsYes,
			},
		},
		qPregnancyPlanning: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aNoPregnancyPlanning,
			},
		},
		qCurrentMedicationsEntry: []*answerTemplate{
			&answerTemplate{
				AnswerText: "Yasmin",
				SubquestionAnswers: map[questionTag][]*answerTemplate{
					qLengthCurrentMedication: []*answerTemplate{
						&answerTemplate{
							AnswerTag: aSixToElevenMonthsLength,
						},
					},
				},
			},
		},
		qInsuranceCoverage: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aInsuranceCoverageNoInsurance,
			},
		},
		qPrevSkinConditionDiagnosis: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aPrevSkinConditionDiagnosisYes,
			},
		},
		qListPrevSkinConditionDiagnosis: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aListPrevSkinConditionDiagnosisAcne,
			},
		},
		qOtherConditionsAcne: []*answerTemplate{
			&answerTemplate{
				AnswerTag: aIntestinalInflammationOtherConditions,
			},
		},
	},
	VisitMessage: "Will my current condition and current medication be considered when prescribing new medications?",
}
