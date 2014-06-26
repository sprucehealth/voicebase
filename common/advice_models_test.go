package common

import (
	"github.com/sprucehealth/backend/encoding"
	"testing"
)

func TestAdviceEquals(t *testing.T) {

	advice1 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			&DoctorInstructionItem{
				Text:     "test1",
				ParentId: encoding.NewObjectId(1),
			},
			&DoctorInstructionItem{
				Text:     "test2",
				ParentId: encoding.NewObjectId(2),
			},
			&DoctorInstructionItem{
				Text:     "test3",
				ParentId: encoding.NewObjectId(3),
			},
			&DoctorInstructionItem{
				Text:     "test4",
				ParentId: encoding.NewObjectId(4),
			},
		},
	}

	advice2 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			&DoctorInstructionItem{
				Text:     "test1",
				ParentId: encoding.NewObjectId(1),
			},
			&DoctorInstructionItem{
				Text:     "test2",
				ParentId: encoding.NewObjectId(2),
			},
			&DoctorInstructionItem{
				Text:     "test3",
				ParentId: encoding.NewObjectId(3),
			},
			&DoctorInstructionItem{
				Text:     "test4",
				ParentId: encoding.NewObjectId(4),
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
			&DoctorInstructionItem{
				Text:     "test1",
				ParentId: encoding.NewObjectId(1),
			},
			&DoctorInstructionItem{
				Text:     "test4",
				ParentId: encoding.NewObjectId(4),
			},
			&DoctorInstructionItem{
				Text:     "test2",
				ParentId: encoding.NewObjectId(2),
			},
			&DoctorInstructionItem{
				Text:     "test3",
				ParentId: encoding.NewObjectId(3),
			},
		},
	}

	advice2 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			&DoctorInstructionItem{
				Text:     "test1",
				ParentId: encoding.NewObjectId(1),
			},
			&DoctorInstructionItem{
				Text:     "test2",
				ParentId: encoding.NewObjectId(2),
			},
			&DoctorInstructionItem{
				Text:     "test3",
				ParentId: encoding.NewObjectId(3),
			},
			&DoctorInstructionItem{
				Text:     "test4",
				ParentId: encoding.NewObjectId(4),
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
			&DoctorInstructionItem{
				Text:     "different text",
				ParentId: encoding.NewObjectId(1),
			},
			&DoctorInstructionItem{
				Text:     "test2",
				ParentId: encoding.NewObjectId(2),
			},
			&DoctorInstructionItem{
				Text:     "test3",
				ParentId: encoding.NewObjectId(3),
			},
			&DoctorInstructionItem{
				Text:     "test4",
				ParentId: encoding.NewObjectId(4),
			},
		},
	}

	advice2 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			&DoctorInstructionItem{
				Text:     "test1",
				ParentId: encoding.NewObjectId(1),
			},
			&DoctorInstructionItem{
				Text:     "test2",
				ParentId: encoding.NewObjectId(2),
			},
			&DoctorInstructionItem{
				Text:     "test3",
				ParentId: encoding.NewObjectId(3),
			},
			&DoctorInstructionItem{
				Text:     "test4",
				ParentId: encoding.NewObjectId(4),
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
			&DoctorInstructionItem{
				Text:     "test1",
				ParentId: encoding.NewObjectId(1),
			},
			&DoctorInstructionItem{
				Text:     "test2",
				ParentId: encoding.NewObjectId(2),
			},
			&DoctorInstructionItem{
				Text:     "test3",
				ParentId: encoding.NewObjectId(3),
			},
			&DoctorInstructionItem{
				Text:     "test4",
				ParentId: encoding.NewObjectId(4),
			},
		},
	}

	advice2 := &Advice{
		SelectedAdvicePoints: []*DoctorInstructionItem{
			&DoctorInstructionItem{
				Text:     "test1",
				ParentId: encoding.NewObjectId(1),
			},
			&DoctorInstructionItem{
				Text:     "test2",
				ParentId: encoding.NewObjectId(2),
			},
			&DoctorInstructionItem{
				Text:     "test3",
				ParentId: encoding.NewObjectId(3),
			},
		},
	}

	if advice1.Equals(advice2) {
		t.Fatalf("expected both advice items to not be equal")
	}
}
