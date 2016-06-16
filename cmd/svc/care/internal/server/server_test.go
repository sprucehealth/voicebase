package server

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/care/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dosespot"
	"github.com/sprucehealth/backend/libs/dosespot/dosespotmock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/layout"
	layoutmock "github.com/sprucehealth/backend/svc/layout/mock"
	"github.com/sprucehealth/backend/svc/media"
	mediamock "github.com/sprucehealth/backend/svc/media/mock"
)

func init() {
	conc.Testing = true
}

func TestCreateCarePlan(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	srv := New(dl, nil, nil, nil, nil, nil, clock.New())

	cpID, err := models.NewCarePlanID()
	test.OK(t, err)

	now := time.Now()

	dl.Expect(mock.NewExpectation(dl.CreateCarePlan, &models.CarePlan{
		Name:         "name",
		CreatorID:    "creator",
		Instructions: []*models.CarePlanInstruction{{Title: "title", Steps: []string{"one", "two"}}},
		Treatments: []*models.CarePlanTreatment{
			{
				MedicationID:         "medicationID",
				EPrescribe:           true,
				Name:                 "name",
				Form:                 "form",
				Route:                "route",
				Availability:         models.TreatmentAvailabilityOTC,
				Dosage:               "dosage",
				DispenseType:         "dispenseType",
				DispenseNumber:       1,
				Refills:              2,
				SubstitutionsAllowed: true,
				DaysSupply:           3,
				Sig:                  "sig",
				PharmacyID:           "pharmacyID",
				PharmacyInstructions: "pharmacyInstructions",
			},
		},
	}).WithReturns(cpID, nil))
	tID, err := models.NewCarePlanTreatmentID()
	test.OK(t, err)
	dl.Expect(mock.NewExpectation(dl.CarePlan, cpID).WithReturns(
		&models.CarePlan{
			ID:           cpID,
			Name:         "name",
			CreatorID:    "creator",
			Instructions: []*models.CarePlanInstruction{{Title: "title", Steps: []string{"one", "two"}}},
			Created:      now,
			Submitted:    &now,
			ParentID:     "pid",
			Treatments: []*models.CarePlanTreatment{
				{
					ID:                   tID,
					MedicationID:         "medicationID",
					EPrescribe:           true,
					Name:                 "name",
					Form:                 "form",
					Route:                "route",
					Availability:         models.TreatmentAvailabilityOTC,
					Dosage:               "dosage",
					DispenseType:         "dispenseType",
					DispenseNumber:       1,
					Refills:              2,
					SubstitutionsAllowed: true,
					DaysSupply:           3,
					Sig:                  "sig",
					PharmacyID:           "pharmacyID",
					PharmacyInstructions: "pharmacyInstructions",
				},
			},
		}, nil))

	req := &care.CreateCarePlanRequest{
		Name:         "name",
		CreatorID:    "creator",
		Instructions: []*care.CarePlanInstruction{{Title: "title", Steps: []string{"one", "two"}}},
		Treatments: []*care.CarePlanTreatment{
			{
				MedicationID:         "medicationID",
				EPrescribe:           true,
				Name:                 "name",
				Form:                 "form",
				Route:                "route",
				Availability:         care.CarePlanTreatment_OTC,
				Dosage:               "dosage",
				DispenseType:         "dispenseType",
				DispenseNumber:       1,
				Refills:              2,
				SubstitutionsAllowed: true,
				DaysSupply:           3,
				Sig:                  "sig",
				PharmacyID:           "pharmacyID",
				PharmacyInstructions: "pharmacyInstructions",
			},
		},
	}
	res, err := srv.CreateCarePlan(nil, req)
	test.OK(t, err)
	test.Equals(t, &care.CreateCarePlanResponse{
		CarePlan: &care.CarePlan{
			ID:                 cpID.String(),
			Name:               "name",
			CreatorID:          "creator",
			Instructions:       []*care.CarePlanInstruction{{Title: "title", Steps: []string{"one", "two"}}},
			CreatedTimestamp:   uint64(now.Unix()),
			SubmittedTimestamp: uint64(now.Unix()),
			Submitted:          true,
			ParentID:           "pid",
			Treatments: []*care.CarePlanTreatment{
				{
					MedicationID:         "medicationID",
					EPrescribe:           true,
					Name:                 "name",
					Form:                 "form",
					Route:                "route",
					Availability:         care.CarePlanTreatment_OTC,
					Dosage:               "dosage",
					DispenseType:         "dispenseType",
					DispenseNumber:       1,
					Refills:              2,
					SubstitutionsAllowed: true,
					DaysSupply:           3,
					Sig:                  "sig",
					PharmacyID:           "pharmacyID",
					PharmacyInstructions: "pharmacyInstructions",
				},
			},
		},
	}, res)
}

func TestSearchMedications(t *testing.T) {
	t.Parallel()
	dsMock := dosespotmock.New(t)
	srv := New(nil, nil, nil, nil, nil, dsMock, clock.New())

	dsMock.Expect(mock.NewExpectation(dsMock.GetDrugNamesForDoctor, int64(123), "tret").WithReturns(
		[]string{"Tretinoin Topical (topical - cream)", "Tretinoin Topical (topical - gel)"}, nil))
	dsMock.Expect(mock.NewExpectation(dsMock.SearchForMedicationStrength, int64(123), "Tretinoin Topical (topical - cream)").WithReturns(
		[]string{"0.025%", "0.05%"}, nil))
	dsMock.Expect(mock.NewExpectation(dsMock.SelectMedication, int64(123), "Tretinoin Topical (topical - cream)", "0.025%").WithReturns(
		&dosespot.MedicationSelectResponse{
			OTC:                     false,
			Schedule:                "0",
			RouteDescription:        "topical",
			DoseFormDescription:     "cream",
			DispenseUnitDescription: "Tube(s)",
			StrengthDescription:     "0.025%",
			GenericProductName:      "tretinoin 0.025% topical cream",
			LexiGenProductID:        1,
			LexiDrugSynID:           2,
			LexiSynonymTypeID:       3,
			RepresentativeNDC:       "111",
		}, nil))
	dsMock.Expect(mock.NewExpectation(dsMock.SelectMedication, int64(123), "Tretinoin Topical (topical - cream)", "0.05%").WithReturns(
		&dosespot.MedicationSelectResponse{
			OTC:                     true,
			Schedule:                "1",
			RouteDescription:        "topical",
			DoseFormDescription:     "cream",
			DispenseUnitDescription: "Tube(s)",
			StrengthDescription:     "0.05%",
			GenericProductName:      "tretinoin 0.05% topical cream",
			LexiGenProductID:        1,
			LexiDrugSynID:           2,
			LexiSynonymTypeID:       3,
			RepresentativeNDC:       "222",
		}, nil))
	dsMock.Expect(mock.NewExpectation(dsMock.SearchForMedicationStrength, int64(123), "Tretinoin Topical (topical - gel)").WithReturns(
		[]string{"0.01%"}, nil))
	dsMock.Expect(mock.NewExpectation(dsMock.SelectMedication, int64(123), "Tretinoin Topical (topical - gel)", "0.01%").WithReturns(
		&dosespot.MedicationSelectResponse{
			OTC:                     false,
			Schedule:                "0",
			RouteDescription:        "topical",
			DoseFormDescription:     "gel",
			DispenseUnitDescription: "Tube(s)",
			StrengthDescription:     "0.01%",
			GenericProductName:      "tretinoin 0.01% topical cream",
			LexiGenProductID:        1,
			LexiDrugSynID:           2,
			LexiSynonymTypeID:       3,
			RepresentativeNDC:       "333",
		}, nil))

	res, err := srv.SearchMedications(nil, &care.SearchMedicationsRequest{
		Query:       "tret",
		ClinicianID: 123,
	})
	test.OK(t, err)
	test.Equals(t, &care.SearchMedicationsResponse{
		Medications: []*care.Medication{
			{
				ID:    "Tretinoin Topical (topical - cream)",
				Name:  "Tretinoin Topical",
				Route: "topical",
				Form:  "cream",
				Strengths: []*care.MedicationStrength{
					{
						OTC:               false,
						Schedule:          0,
						DispenseUnit:      "Tube(s)",
						Strength:          "0.025%",
						GenericName:       "tretinoin",
						LexiGenProductID:  1,
						LexiDrugSynID:     2,
						LexiSynonymTypeID: 3,
						NDC:               "111",
					},
					{
						OTC:               true,
						Schedule:          1,
						DispenseUnit:      "Tube(s)",
						Strength:          "0.05%",
						GenericName:       "tretinoin",
						LexiGenProductID:  1,
						LexiDrugSynID:     2,
						LexiSynonymTypeID: 3,
						NDC:               "222",
					},
				},
			},
			{
				ID:    "Tretinoin Topical (topical - gel)",
				Name:  "Tretinoin Topical",
				Route: "topical",
				Form:  "gel",
				Strengths: []*care.MedicationStrength{
					{
						OTC:               false,
						Schedule:          0,
						DispenseUnit:      "Tube(s)",
						Strength:          "0.01%",
						GenericName:       "tretinoin",
						LexiGenProductID:  1,
						LexiDrugSynID:     2,
						LexiSynonymTypeID: 3,
						NDC:               "333",
					},
				},
			},
		},
	}, res)
}

func TestSelfReportedMedicationsSearch(t *testing.T) {
	t.Parallel()
	dsMock := dosespotmock.New(t)
	srv := New(nil, nil, nil, nil, nil, dsMock, clock.New())

	dsMock.Expect(mock.NewExpectation(dsMock.GetDrugNamesForPatient, "Advil").WithReturns(
		[]string{
			"Advil 1",
			"Advil 2",
			"Advil 3",
		}, nil))

	res, err := srv.SearchSelfReportedMedications(context.Background(), &care.SearchSelfReportedMedicationsRequest{
		Query: "Advil",
	})

	test.OK(t, err)
	test.Equals(t, &care.SearchSelfReportedMedicationsResponse{
		Results: []string{
			"Advil 1",
			"Advil 2",
			"Advil 3",
		},
	}, res)
}

func TestAllergyMedicationsSearch(t *testing.T) {
	t.Parallel()
	dsMock := dosespotmock.New(t)
	srv := New(nil, nil, nil, nil, nil, dsMock, clock.New())

	dsMock.Expect(mock.NewExpectation(dsMock.SearchForAllergyRelatedMedications, "Advil").WithReturns(
		[]string{
			"Advil 1",
			"Advil 2",
			"Advil 3",
		}, nil))

	res, err := srv.SearchAllergyMedications(context.Background(), &care.SearchAllergyMedicationsRequest{
		Query: "Advil",
	})

	test.OK(t, err)
	test.Equals(t, &care.SearchAllergyMedicationsResponse{
		Results: []string{
			"Advil 1",
			"Advil 2",
			"Advil 3",
		},
	}, res)
}

func TestTriageVisit(t *testing.T) {
	t.Parallel()
	dalMock := dalmock.New(t)
	defer dalMock.Finish()

	mclk := clock.NewManaged(time.Now())

	visitID, err := models.NewVisitID()
	test.OK(t, err)

	dalMock.Expect(mock.NewExpectation(dalMock.UpdateVisit, visitID, &dal.VisitUpdate{
		Triaged:     ptr.Bool(true),
		TriagedTime: ptr.Time(mclk.Now()),
	}).WithReturns(int64(1), nil))

	srv := New(dalMock, nil, nil, nil, nil, nil, mclk)

	_, err = srv.TriageVisit(context.Background(), &care.TriageVisitRequest{
		VisitID: visitID.String(),
	})
	test.OK(t, err)
}

func TestCreateVisitAnswers_MediaSection(t *testing.T) {
	t.Parallel()
	dalMock := dalmock.New(t)
	layoutMock := layoutmock.New(t)
	layoutStorageMock := layoutmock.NewStore(t)
	mediaMock := mediamock.New(t)
	defer dalMock.Finish()
	defer layoutMock.Finish()
	defer layoutStorageMock.Finish()
	defer mediaMock.Finish()

	mclk := clock.NewManaged(time.Now())

	visitID, err := models.NewVisitID()
	test.OK(t, err)

	clientAnswersJSON := `{
	  "answers": {
	    "1:q_photo_face_question_id": {
	      "type": "q_type_media_section",
	      "sections": [
	        {
	          "name": "Testing",
	          "media": [
	            {
	              "name": "Other Location",
	              "slot_id": "7901",
	              "media_id": "7966"
	            },
	            {
	              "name": "Other Location",
	              "slot_id": "7902",
	              "media_id": "7967"
	            }
	          ]
	        }
	      ]
	    }
		}
}`

	dalMock.Expect(mock.NewExpectation(dalMock.Visit, visitID).WithReturns(&models.Visit{
		LayoutVersionID: "layoutVersionID",
		ID:              visitID,
	}, nil))

	layoutMock.Expect(mock.NewExpectation(layoutMock.GetVisitLayoutVersion, &layout.GetVisitLayoutVersionRequest{
		ID: "layoutVersionID",
	}).WithReturns(&layout.GetVisitLayoutVersionResponse{
		VisitLayoutVersion: &layout.VisitLayoutVersion{
			IntakeLayoutLocation: "layoutLocation",
		},
	}, nil))

	layoutStorageMock.Expect(mock.NewExpectation(layoutStorageMock.GetIntake, "layoutLocation").WithReturns(&layout.Intake{
		Sections: []*layout.Section{
			{
				Screens: []*layout.Screen{
					{
						Questions: []*layout.Question{
							{
								ID:   "1:q_photo_face_question_id",
								Type: layout.QuestionTypeMediaSection,
								MediaSlots: []*layout.MediaSlot{
									{
										Type: "video",
										ID:   "7901",
									},
									{
										Type: "video",
										ID:   "7902",
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil))

	mediaMock.Expect(mock.NewExpectation(mediaMock.ClaimMedia, &media.ClaimMediaRequest{
		MediaIDs:  []string{"7966", "7967"},
		OwnerType: media.MediaOwnerType_VISIT,
		OwnerID:   visitID.String(),
	}))

	mediaMock.Expect(mock.NewExpectation(mediaMock.MediaInfos, &media.MediaInfosRequest{
		MediaIDs: []string{"7966", "7967"},
	}).WithReturns(&media.MediaInfosResponse{
		MediaInfos: map[string]*media.MediaInfo{
			"7966": {
				ID: "7966",
				MIME: &media.MIME{
					Type: "video",
				},
			},
			"7967": {
				ID: "7967",
				MIME: &media.MIME{
					Type: "video",
				},
			},
		},
	}, nil))

	dalMock.Expect(mock.NewExpectation(dalMock.CreateVisitAnswer, visitID, "entityID", &models.Answer{
		QuestionID: "1:q_photo_face_question_id",
		Type:       layout.QuestionTypeMediaSection,
		Answer: &models.Answer_MediaSection{
			MediaSection: &models.MediaSectionAnswer{
				Sections: []*models.MediaSectionAnswer_MediaSectionItem{
					{
						Name: "Testing",
						Slots: []*models.MediaSectionAnswer_MediaSectionItem_MediaSlotItem{
							{
								Name:    "Other Location",
								SlotID:  "7901",
								MediaID: "7966",
								Type:    models.MediaType_VIDEO,
							},
							{
								Name:    "Other Location",
								SlotID:  "7902",
								MediaID: "7967",
								Type:    models.MediaType_VIDEO,
							},
						},
					},
				},
			},
		},
	}))

	srv := New(dalMock, layoutMock, nil, mediaMock, layoutStorageMock, nil, mclk)

	_, err = srv.CreateVisitAnswers(context.Background(), &care.CreateVisitAnswersRequest{
		VisitID:       visitID.String(),
		AnswersJSON:   clientAnswersJSON,
		ActorEntityID: "entityID",
	})
	test.OK(t, err)
}
