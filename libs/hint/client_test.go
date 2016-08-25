package hint

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestPatientClient(t *testing.T) {
	Testing = true
	practiceKey := os.Getenv("HINT_PRACTICE_KEY")
	if practiceKey == "" {
		t.Skip("Skipping tests since practice key not found")
	}

	t.Run("CreateSingle", func(t *testing.T) {
		patient, err := NewPatient(practiceKey, &PatientParams{
			FirstName: fmt.Sprintf("Test%d", time.Now().UnixNano()),
			LastName:  "Test",
		})
		if err != nil {
			t.Fatal(err)
		}

		patient2, err := GetPatient(practiceKey, patient.ID)
		if err != nil {
			t.Fatal(err)
		}

		// delete the patient that was created
		if err := DeletePatient(practiceKey, patient.ID); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(patient, patient2) {
			t.Fatalf("Expected the patient \n%#v to be equal to \n%#v but wasnt", patient, patient2)
		}
	})

	t.Run("Iterate", func(t *testing.T) {
		// create 5 threads
		name := fmt.Sprintf("%d", time.Now().UnixNano())
		for i := 0; i < 5; i++ {
			_, err := NewPatient(practiceKey, &PatientParams{
				FirstName: name,
				LastName:  "Test",
			})
			if err != nil {
				t.Fatal(err)
			}
		}

		iter := ListPatient(practiceKey, &ListParams{
			Items: []*QueryItem{
				{
					Field: "first_name",
					Operations: []*Operation{
						{
							Operator: OperatorEqualTo,
							Operand:  name,
						},
					},
				},
			},
		})

		// should get back 5 items
		patientIDs := make(map[string]struct{})
		for iter.Next() {
			if err := iter.Err(); err != nil {
				t.Fatal(err)
			}
			patientIDs[iter.Current().(*Patient).ID] = struct{}{}
		}
		if len(patientIDs) != 5 {
			t.Fatalf("Expected 5 patients to have been created but only %d were created", len(patientIDs))
		}

		// delete all threads created
		for patientID := range patientIDs {
			if err := DeletePatient(practiceKey, patientID); err != nil {
				t.Fatal(err)
			}
		}
	})

	t.Run("IterateTime", func(t *testing.T) {
		startTime := time.Now()
		name := fmt.Sprintf("%d", time.Now().UnixNano())

		// create 3 threads, 3 seconds apart
		for i := 0; i < 3; i++ {
			_, err := NewPatient(practiceKey, &PatientParams{
				FirstName: name,
				LastName:  "Test",
			})
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(time.Second)
		}

		// query for 2 threads created in the first 2 seconds
		// to ensure that the date filtering is working as expected
		iter := ListPatient(practiceKey, &ListParams{
			Items: []*QueryItem{
				{
					Field: "created_at",
					Operations: []*Operation{
						{
							Operator: OperatorGreaterThanEqualTo,
							Operand:  startTime.String(),
						},
						{
							Operator: OperatorLessThan,
							Operand:  startTime.Add(3 * time.Second).String(),
						},
					},
				},
			},
			Sort: &Sort{
				By: "created_at",
			},
		})

		// ensure that we only get back 2 threads
		patientIDs := make(map[string]struct{})
		for iter.Next() {
			if err := iter.Err(); err != nil {
				t.Fatal(err)
			}
			patientIDs[iter.Current().(*Patient).ID] = struct{}{}
		}
		if len(patientIDs) != 2 {
			t.Fatalf("Expected 2 patients to have been created but only %d were created", len(patientIDs))
		}

		// delete all the threads just created
		iter = ListPatient(practiceKey, &ListParams{
			Items: []*QueryItem{
				{
					Field: "first_name",
					Operations: []*Operation{
						{
							Operator: OperatorEqualTo,
							Operand:  name,
						},
					},
				},
			},
		})
		patientIDs = make(map[string]struct{})
		for iter.Next() {
			if err := iter.Err(); err != nil {
				t.Fatal(err)
			}
			patientIDs[iter.Current().(*Patient).ID] = struct{}{}
		}
		for patientID := range patientIDs {
			if err := DeletePatient(practiceKey, patientID); err != nil {
				t.Fatal(err)
			}
		}
	})

	t.Run("Update", func(t *testing.T) {
		patient, err := NewPatient(practiceKey, &PatientParams{
			FirstName: fmt.Sprintf("Test%d", time.Now().UnixNano()),
			LastName:  "Test",
		})
		if err != nil {
			t.Fatal(err)
		}

		updatedPatient, err := UpdatePatient(practiceKey, patient.ID, &PatientParams{
			FirstName: patient.FirstName,
			LastName:  patient.LastName,
			Phones: []*Phone{
				{
					Type:   "Mobile",
					Number: "7348465522",
				},
				{
					Type:   "Mobile",
					Number: "2068773590",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		// updated patient should have 2 phone numbers
		if !reflect.DeepEqual([]*Phone{{
			Type:   "Mobile",
			Number: "7348465522",
		},
			{
				Type:   "Mobile",
				Number: "2068773590",
			},
		}, updatedPatient.Phones) {
			t.Fatal("Expected phone numbers for the patient to be updated but they werent")
		}

		// delete created patient
		if err := DeletePatient(practiceKey, patient.ID); err != nil {
			t.Fatal(err)
		}
	})
}
