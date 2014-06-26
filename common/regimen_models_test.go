package common

import (
	"github.com/sprucehealth/backend/encoding"
	"testing"
)

func TestRegimenPlanEquals(t *testing.T) {

	// test simple regimen plan to be equal
	// even include items in there with no parent id
	regimenPlan1 := &RegimenPlan{
		RegimenSections: []*RegimenSection{
			&RegimenSection{
				RegimenName: "testRegimen1",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test1",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1b",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1c",
						ParentId: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				RegimenName: "testRegimen2",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test2",
						ParentId: encoding.NewObjectId(2),
					},
					&DoctorInstructionItem{
						Text: "test2a",
					},
				},
			},
		},
	}

	regimenPlan2 := &RegimenPlan{
		RegimenSections: []*RegimenSection{
			&RegimenSection{
				RegimenName: "testRegimen1",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test1",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1b",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1c",
						ParentId: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				RegimenName: "testRegimen2",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test2",
						ParentId: encoding.NewObjectId(2),
					},
					&DoctorInstructionItem{
						Text: "test2a",
					},
				},
			},
		},
	}

	if !regimenPlan1.Equals(regimenPlan2) {
		t.Fatal("Expcted both regimen plans to be equal")
	}

}

func TestRegimenPlanEquals_EmptyTest(t *testing.T) {
	var reg1, reg2 *RegimenPlan
	if !reg1.Equals(reg2) {
		t.Fatalf("Expected nil regimen plans to be equal")
	}

	// test empty regimen plans
	regimenPlan1 := &RegimenPlan{}
	regimenPlan2 := &RegimenPlan{}

	if !regimenPlan1.Equals(regimenPlan2) {
		t.Fatalf("Expected the regimen plans to be equal")
	}
}

func TestRegimenPlanEquals_DifferentOrderTest(t *testing.T) {
	// test simple regimen plan to be unequal when the ordering
	// of steps is unequal
	regimenPlan1 := &RegimenPlan{
		RegimenSections: []*RegimenSection{
			&RegimenSection{
				RegimenName: "testRegimen1",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test1b",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1c",
						ParentId: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				RegimenName: "testRegimen2",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test2",
						ParentId: encoding.NewObjectId(2),
					},
					&DoctorInstructionItem{
						Text:     "test2a",
						ParentId: encoding.NewObjectId(1),
					},
				},
			},
		},
	}

	regimenPlan2 := &RegimenPlan{
		RegimenSections: []*RegimenSection{
			&RegimenSection{
				RegimenName: "testRegimen1",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test1",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1b",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1c",
						ParentId: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				RegimenName: "testRegimen2",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test2",
						ParentId: encoding.NewObjectId(2),
					},
					&DoctorInstructionItem{
						Text:     "test2a",
						ParentId: encoding.NewObjectId(1),
					},
				},
			},
		},
	}

	if regimenPlan1.Equals(regimenPlan2) {
		t.Fatal("Expcted both regimen plans to be equal")
	}
}

func TestRegimenPlanEquals_DifferentSectionNamesTest(t *testing.T) {

	regimenPlan1 := &RegimenPlan{
		RegimenSections: []*RegimenSection{
			&RegimenSection{
				RegimenName: "different name for testRegimen1",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test1",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1b",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1c",
						ParentId: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				RegimenName: "testRegimen2",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test2",
						ParentId: encoding.NewObjectId(2),
					},
					&DoctorInstructionItem{
						Text: "test2a",
					},
				},
			},
		},
	}

	regimenPlan2 := &RegimenPlan{
		RegimenSections: []*RegimenSection{
			&RegimenSection{
				RegimenName: "testRegimen1",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test1",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1b",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1c",
						ParentId: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				RegimenName: "testRegimen2",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test2",
						ParentId: encoding.NewObjectId(2),
					},
					&DoctorInstructionItem{
						Text: "test2a",
					},
				},
			},
		},
	}

	if regimenPlan1.Equals(regimenPlan2) {
		t.Fatal("Expcted both regimen plans to be equal")
	}
}

// In this test we are testing to ensure that if one of the regimen plans
// has an empty regimen section while the rest of the filled regimen sections are
// identical in content, then they are considered equal
func TestRegimenPlanEquals_DifferentEmptySectionsTest(t *testing.T) {

	regimenPlan1 := &RegimenPlan{
		RegimenSections: []*RegimenSection{
			&RegimenSection{
				RegimenName: "testRegimen1",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test1",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1b",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1c",
						ParentId: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				RegimenName: "testRegimen2",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test2",
						ParentId: encoding.NewObjectId(2),
					},
					&DoctorInstructionItem{
						Text: "test2a",
					},
				},
			},
			&RegimenSection{
				RegimenName: "nilRegimenSection",
			},
		},
	}

	regimenPlan2 := &RegimenPlan{
		RegimenSections: []*RegimenSection{
			&RegimenSection{
				RegimenName: "testRegimen1",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test1",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1b",
						ParentId: encoding.NewObjectId(1),
					},
					&DoctorInstructionItem{
						Text:     "test1c",
						ParentId: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				RegimenName: "testRegimen2",
				RegimenSteps: []*DoctorInstructionItem{
					&DoctorInstructionItem{
						Text:     "test2",
						ParentId: encoding.NewObjectId(2),
					},
					&DoctorInstructionItem{
						Text: "test2a",
					},
				},
			},
		},
	}

	if !regimenPlan1.Equals(regimenPlan2) {
		t.Fatal("Expected both regimen plans to be equal")
	}
}
