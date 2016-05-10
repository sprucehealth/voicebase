package server

import (
	"testing"
	"time"

	dalmock "github.com/sprucehealth/backend/cmd/svc/care/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/test"
)

func TestCreateCarePlan(t *testing.T) {
	t.Parallel()
	dl := dalmock.New(t)
	defer dl.Finish()
	srv := New(dl, nil, nil)

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
