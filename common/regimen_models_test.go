package common

import (
	"testing"

	"github.com/sprucehealth/backend/encoding"
)

func TestRegimenPlanEquals(t *testing.T) {

	// test simple regimen plan to be equal
	// even include items in there with no parent id
	regimenPlan1 := &RegimenPlan{
		Sections: []*RegimenSection{
			&RegimenSection{
				Name: "testRegimen1",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test1",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1b",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1c",
						ParentID: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				Name: "testRegimen2",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test2",
						ParentID: encoding.NewObjectId(2),
					},
					{
						Text: "test2a",
					},
				},
			},
		},
	}

	regimenPlan2 := &RegimenPlan{
		Sections: []*RegimenSection{
			&RegimenSection{
				Name: "testRegimen1",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test1",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1b",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1c",
						ParentID: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				Name: "testRegimen2",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test2",
						ParentID: encoding.NewObjectId(2),
					},
					{
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
		Sections: []*RegimenSection{
			&RegimenSection{
				Name: "testRegimen1",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test1b",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1c",
						ParentID: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				Name: "testRegimen2",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test2",
						ParentID: encoding.NewObjectId(2),
					},
					{
						Text:     "test2a",
						ParentID: encoding.NewObjectId(1),
					},
				},
			},
		},
	}

	regimenPlan2 := &RegimenPlan{
		Sections: []*RegimenSection{
			&RegimenSection{
				Name: "testRegimen1",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test1",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1b",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1c",
						ParentID: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				Name: "testRegimen2",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test2",
						ParentID: encoding.NewObjectId(2),
					},
					{
						Text:     "test2a",
						ParentID: encoding.NewObjectId(1),
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
		Sections: []*RegimenSection{
			&RegimenSection{
				Name: "different name for testRegimen1",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test1",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1b",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1c",
						ParentID: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				Name: "testRegimen2",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test2",
						ParentID: encoding.NewObjectId(2),
					},
					{
						Text: "test2a",
					},
				},
			},
		},
	}

	regimenPlan2 := &RegimenPlan{
		Sections: []*RegimenSection{
			&RegimenSection{
				Name: "testRegimen1",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test1",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1b",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1c",
						ParentID: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				Name: "testRegimen2",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test2",
						ParentID: encoding.NewObjectId(2),
					},
					{
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
		Sections: []*RegimenSection{
			&RegimenSection{
				Name: "testRegimen1",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test1",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1b",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1c",
						ParentID: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				Name: "testRegimen2",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test2",
						ParentID: encoding.NewObjectId(2),
					},
					{
						Text: "test2a",
					},
				},
			},
			&RegimenSection{
				Name: "nilRegimenSection",
			},
		},
	}

	regimenPlan2 := &RegimenPlan{
		Sections: []*RegimenSection{
			&RegimenSection{
				Name: "testRegimen1",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test1",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1b",
						ParentID: encoding.NewObjectId(1),
					},
					{
						Text:     "test1c",
						ParentID: encoding.NewObjectId(1),
					},
				},
			},
			&RegimenSection{
				Name: "testRegimen2",
				Steps: []*DoctorInstructionItem{
					{
						Text:     "test2",
						ParentID: encoding.NewObjectId(2),
					},
					{
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
