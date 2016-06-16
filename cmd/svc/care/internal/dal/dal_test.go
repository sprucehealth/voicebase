package dal

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testsql"
)

const schemaGlob = "schema/v*.sql"

func TestCarePlan(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)

	id, err := models.NewCarePlanID()
	test.OK(t, err)
	_, err = dal.CarePlan(nil, id)
	test.Equals(t, ErrNotFound, errors.Cause(err))

	err = dal.SubmitCarePlan(nil, id, "parent2")
	test.Equals(t, ErrNotFound, errors.Cause(err))

	expCP := &models.CarePlan{
		Name:      "name",
		CreatorID: "creator",
		Instructions: []*models.CarePlanInstruction{
			{
				Title: "Morning",
				Steps: []string{"one", "two"},
			},
			{
				Title: "Night",
				Steps: []string{"123", "xyz"},
			},
		},
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
	}
	id, err = dal.CreateCarePlan(nil, expCP)
	test.OK(t, err)
	test.Equals(t, true, id.IsValid)

	cp, err := dal.CarePlan(nil, id)
	test.OK(t, err)
	expCP.Created = cp.Created
	test.Equals(t, expCP, cp)

	test.OK(t, dal.SubmitCarePlan(nil, id, "parent"))
	cp, err = dal.CarePlan(nil, id)
	test.OK(t, err)
	test.Assert(t, cp.Submitted != nil, "Submitted should not be nil")
	test.Equals(t, "parent", cp.ParentID)

	err = dal.SubmitCarePlan(nil, id, "parent2")
	test.Equals(t, ErrAlreadySubmitted, errors.Cause(err))
	cp, err = dal.CarePlan(nil, id)
	test.OK(t, err)
	test.Assert(t, cp.Submitted != nil, "Submitted should not be nil")
	test.Equals(t, "parent", cp.ParentID)
}
