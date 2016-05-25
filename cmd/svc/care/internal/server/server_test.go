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
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/test"
)

func init() {
	conc.Testing = true
}

func TestCreateCarePlan(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	srv := New(dl, nil, nil, nil, nil, clock.New())

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
	srv := New(nil, nil, nil, nil, dsMock, clock.New())

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
	srv := New(nil, nil, nil, nil, dsMock, clock.New())

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
	srv := New(nil, nil, nil, nil, dsMock, clock.New())

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

	srv := New(dalMock, nil, nil, nil, nil, mclk)

	_, err = srv.TriageVisit(context.Background(), &care.TriageVisitRequest{
		VisitID: visitID.String(),
	})
	test.OK(t, err)
}