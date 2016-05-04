package layout

import (
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/test"

	"testing"
)

func TestSAML(t *testing.T) {
	store := NewStore(storage.NewTestStore(nil))
	testSAML(t, "test1", store)
	testSAML(t, "test2", store)
}

func testSAML(t *testing.T, name string, store Storage) {
	testSAML := "testSAML"
	samlLocation, err := store.PutSAML(name, testSAML)
	if err != nil {
		t.Fatalf(err.Error())
	}

	testSAMLAfterRead, err := store.GetSAML(samlLocation)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if testSAML != testSAMLAfterRead {
		t.Fatalf("Expected %s, got %s", testSAML, testSAMLAfterRead)
	}
}

func TestIntake(t *testing.T) {
	store := NewStore(storage.NewTestStore(nil))

	testIntake := &Intake{
		Header: &Header{
			Title: "Hello",
		},
	}

	intakeLocation, err := store.PutIntake("test", testIntake)
	if err != nil {
		t.Fatalf(err.Error())
	}

	testIntakeAfterRead, err := store.GetIntake(intakeLocation)
	if err != nil {
		t.Fatalf(err.Error())
	}

	test.Equals(t, testIntake, testIntakeAfterRead)
}

func TestReview(t *testing.T) {
	store := NewStore(storage.NewTestStore(nil))

	testReview := &visitreview.SectionListView{
		Sections: []visitreview.View{
			&visitreview.TitleSubtitleLabels{
				Title: "SUP",
			},
		},
	}
	if err := testReview.Validate(); err != nil {
		t.Fatal(err)
	}

	reviewLocation, err := store.PutReview("test", testReview)
	if err != nil {
		t.Fatalf(err.Error())
	}

	testReviewAfterRead, err := store.GetReview(reviewLocation)
	if err != nil {
		t.Fatalf(err.Error())
	}

	test.Equals(t, testReview, testReviewAfterRead)
}
