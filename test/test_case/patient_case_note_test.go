package test_case

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/patient_case/model"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestPatientCaseNoteInteraction(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	dr := test_integration.SignupRandomTestDoctorInState("PA", t, testData)
	pv := test_integration.CreateRandomPatientVisitInState("PA", t, testData)
	pCase, err := testData.DataAPI.GetPatientCaseFromPatientVisitID(pv.PatientVisitID)
	test.OK(t, err)

	// Insert a case note record
	noteText := "Test Content"
	noteID, err := testData.DataAPI.InsertPatientCaseNote(&model.PatientCaseNote{
		CaseID:         pCase.ID.Int64(),
		AuthorDoctorID: dr.DoctorID,
		NoteText:       noteText,
	})
	test.OK(t, err)
	test.Assert(t, noteID != 0, "Expected non zero id to be returned")

	// Fetch the same note back and verify round trip contents
	note, err := testData.DataAPI.PatientCaseNote(noteID)
	test.OK(t, err)
	test.Equals(t, pCase.ID.Int64(), note.CaseID)
	test.Equals(t, dr.DoctorID, note.AuthorDoctorID)
	test.Equals(t, noteText, note.NoteText)
	test.Equals(t, noteID, note.ID)
	test.Assert(t, !note.Created.IsZero(), "Expected a non zero creation timestamp")
	test.Assert(t, !note.Modified.IsZero(), "Expected a non zero modified timestamp")

	// This is ugly but assert our modified update schema with a sleep here
	time.Sleep(1 * time.Second)

	// Update the record and assert modified changes and text text if reflected
	oldModified := note.Modified
	noteText = "New Test Content"
	aff, err := testData.DataAPI.UpdatePatientCaseNote(&model.PatientCaseNoteUpdate{
		ID:       note.ID,
		NoteText: noteText,
	})
	test.OK(t, err)
	test.Equals(t, int64(1), aff)

	// Read our contents back and assert the update
	note, err = testData.DataAPI.PatientCaseNote(noteID)
	test.OK(t, err)
	test.Equals(t, pCase.ID.Int64(), note.CaseID)
	test.Equals(t, dr.DoctorID, note.AuthorDoctorID)
	test.Equals(t, noteText, string(note.NoteText))
	test.Equals(t, noteID, note.ID)
	test.Assert(t, !note.Created.IsZero(), "Expected a non zero creation timestamp")
	test.Assert(t, !note.Modified.IsZero(), "Expected a non zero modified timestamp")
	test.Assert(t, note.Modified.Unix() != oldModified.Unix(), "Expected the updated time to have changed")

	// Create 2 more notes
	noteText2 := "Test Content3"
	noteID2, err := testData.DataAPI.InsertPatientCaseNote(&model.PatientCaseNote{
		CaseID:         pCase.ID.Int64(),
		AuthorDoctorID: dr.DoctorID,
		NoteText:       noteText2,
	})
	test.OK(t, err)
	test.Assert(t, noteID != 0, "Expected non zero id to be returned")

	noteText3 := "Test Content3"
	noteID3, err := testData.DataAPI.InsertPatientCaseNote(&model.PatientCaseNote{
		CaseID:         pCase.ID.Int64(),
		AuthorDoctorID: dr.DoctorID,
		NoteText:       noteText3,
	})
	test.OK(t, err)
	test.Assert(t, noteID != 0, "Expected non zero id to be returned")

	// Test the empty list bulk get case
	notes, err := testData.DataAPI.PatientCaseNotes(nil)
	test.OK(t, err)
	test.Equals(t, 0, len(notes))

	// Get our records in bulk
	notes, err = testData.DataAPI.PatientCaseNotes([]int64{pCase.ID.Int64()})
	test.OK(t, err)
	test.Equals(t, 1, len(notes))
	test.Equals(t, 3, len(notes[pCase.ID.Int64()]))

	// We should be able to assert our order as well
	// 0
	test.Equals(t, pCase.ID.Int64(), notes[pCase.ID.Int64()][0].CaseID)
	test.Equals(t, dr.DoctorID, notes[pCase.ID.Int64()][0].AuthorDoctorID)
	test.Equals(t, noteText, notes[pCase.ID.Int64()][0].NoteText)
	test.Equals(t, noteID, notes[pCase.ID.Int64()][0].ID)
	test.Assert(t, !notes[pCase.ID.Int64()][0].Created.IsZero(), "Expected a non zero creation timestamp")
	test.Assert(t, !notes[pCase.ID.Int64()][0].Modified.IsZero(), "Expected a non zero modified timestamp")

	// 1
	test.Equals(t, pCase.ID.Int64(), notes[pCase.ID.Int64()][1].CaseID)
	test.Equals(t, dr.DoctorID, notes[pCase.ID.Int64()][1].AuthorDoctorID)
	test.Equals(t, noteText2, notes[pCase.ID.Int64()][1].NoteText)
	test.Equals(t, noteID2, notes[pCase.ID.Int64()][1].ID)
	test.Assert(t, !notes[pCase.ID.Int64()][1].Created.IsZero(), "Expected a non zero creation timestamp")
	test.Assert(t, !notes[pCase.ID.Int64()][1].Modified.IsZero(), "Expected a non zero modified timestamp")

	// 2
	test.Equals(t, pCase.ID.Int64(), notes[pCase.ID.Int64()][2].CaseID)
	test.Equals(t, dr.DoctorID, notes[pCase.ID.Int64()][2].AuthorDoctorID)
	test.Equals(t, noteText3, notes[pCase.ID.Int64()][2].NoteText)
	test.Equals(t, noteID3, notes[pCase.ID.Int64()][2].ID)
	test.Assert(t, !notes[pCase.ID.Int64()][2].Created.IsZero(), "Expected a non zero creation timestamp")
	test.Assert(t, !notes[pCase.ID.Int64()][2].Modified.IsZero(), "Expected a non zero modified timestamp")
}
