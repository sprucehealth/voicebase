package common

import (
	"testing"

	"github.com/sprucehealth/backend/encoding"
)

func TestAdviceEquals(t *testing.T) {

	advice1 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			{
				Text:     "test1",
				ParentID: encoding.NewObjectId(1),
			},
			{
				Text:     "test2",
				ParentID: encoding.NewObjectId(2),
			},
			{
				Text:     "test3",
				ParentID: encoding.NewObjectId(3),
			},
			{
				Text:     "test4",
				ParentID: encoding.NewObjectId(4),
			},
		},
	}

	advice2 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			{
				Text:     "test1",
				ParentID: encoding.NewObjectId(1),
			},
			{
				Text:     "test2",
				ParentID: encoding.NewObjectId(2),
			},
			{
				Text:     "test3",
				ParentID: encoding.NewObjectId(3),
			},
			{
				Text:     "test4",
				ParentID: encoding.NewObjectId(4),
			},
		},
	}

	if !advice1.Equals(advice2) {
		t.Fatalf("expected both advice items to be equal")
	}
}

func TestAdviceEquals_EmptyTest(t *testing.T) {
	var advice1, advice2 *Advice

	if !advice1.Equals(advice2) {
		t.Fatalf("advice1 and advice2 expected to be equal")
	}

	advice1 = &Advice{}
	advice2 = &Advice{}

	if !advice1.Equals(advice2) {
		t.Fatalf("advice1 and advice2 expected to be equal")
	}
}

func TestAdviceEquals_DifferentOrder(t *testing.T) {
	advice1 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			{
				Text:     "test1",
				ParentID: encoding.NewObjectId(1),
			},
			{
				Text:     "test4",
				ParentID: encoding.NewObjectId(4),
			},
			{
				Text:     "test2",
				ParentID: encoding.NewObjectId(2),
			},
			{
				Text:     "test3",
				ParentID: encoding.NewObjectId(3),
			},
		},
	}

	advice2 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			{
				Text:     "test1",
				ParentID: encoding.NewObjectId(1),
			},
			{
				Text:     "test2",
				ParentID: encoding.NewObjectId(2),
			},
			{
				Text:     "test3",
				ParentID: encoding.NewObjectId(3),
			},
			{
				Text:     "test4",
				ParentID: encoding.NewObjectId(4),
			},
		},
	}

	if advice1.Equals(advice2) {
		t.Fatalf("expected both advice items to not be equal")
	}
}

func TestAdviceEquals_DifferentText(t *testing.T) {
	advice1 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			{
				Text:     "different text",
				ParentID: encoding.NewObjectId(1),
			},
			{
				Text:     "test2",
				ParentID: encoding.NewObjectId(2),
			},
			{
				Text:     "test3",
				ParentID: encoding.NewObjectId(3),
			},
			{
				Text:     "test4",
				ParentID: encoding.NewObjectId(4),
			},
		},
	}

	advice2 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			{
				Text:     "test1",
				ParentID: encoding.NewObjectId(1),
			},
			{
				Text:     "test2",
				ParentID: encoding.NewObjectId(2),
			},
			{
				Text:     "test3",
				ParentID: encoding.NewObjectId(3),
			},
			{
				Text:     "test4",
				ParentID: encoding.NewObjectId(4),
			},
		},
	}

	if advice1.Equals(advice2) {
		t.Fatalf("expected both advice items to not be equal")
	}
}

func TestAdviceEquals_DifferentLengths(t *testing.T) {

	advice1 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			{
				Text:     "test1",
				ParentID: encoding.NewObjectId(1),
			},
			{
				Text:     "test2",
				ParentID: encoding.NewObjectId(2),
			},
			{
				Text:     "test3",
				ParentID: encoding.NewObjectId(3),
			},
			{
				Text:     "test4",
				ParentID: encoding.NewObjectId(4),
			},
		},
	}

	advice2 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			{
				Text:     "test1",
				ParentID: encoding.NewObjectId(1),
			},
			{
				Text:     "test2",
				ParentID: encoding.NewObjectId(2),
			},
			{
				Text:     "test3",
				ParentID: encoding.NewObjectId(3),
			},
		},
	}

	if advice1.Equals(advice2) {
		t.Fatalf("expected both advice items to not be equal")
	}
}
