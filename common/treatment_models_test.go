package common

import (
	"testing"

	"github.com/sprucehealth/backend/encoding"
)

func TestTreatmentEquals(t *testing.T) {

	treatment1 := &Treatment{
		ID: encoding.NewObjectID(5),
		DrugDBIDs: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName: "Testing",
		DrugName:         "Testing",
		DosageStrength:   "50mg",
		DispenseValue:    12,
		DispenseUnitID:   encoding.NewObjectID(12),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		}, SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		}, ERx: &ERxData{
			PrescriptionID:     encoding.NewObjectID(20),
			ErxMedicationID:    encoding.NewObjectID(25),
			PrescriptionStatus: "eRxSent",
		},
	}

	treatment2 := &Treatment{
		ID: encoding.NewObjectID(5),
		DrugDBIDs: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName: "Testing",
		DrugName:         "Testing",
		DosageStrength:   "50mg",
		DispenseValue:    12,
		DispenseUnitID:   encoding.NewObjectID(12),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		}, SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		}, ERx: &ERxData{
			PrescriptionID:     encoding.NewObjectID(20),
			ErxMedicationID:    encoding.NewObjectID(25),
			PrescriptionStatus: "eRxSent",
		},
	}

	// treatment1 and treatment 2 should be equal
	if !treatment1.Equals(treatment2) {
		t.Fatal("treatment 1 and 2 expected to be equal")
	}

	treatment3 := &Treatment{
		ID: encoding.NewObjectID(5),
		DrugDBIDs: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName: "Testing",
		DrugName:         "Testing",
		DosageStrength:   "50mg",
		DispenseValue:    12,
		DispenseUnitID:   encoding.NewObjectID(12),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		}, SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		}, ERx: &ERxData{
			PrescriptionID:     encoding.NewObjectID(21),
			ErxMedicationID:    encoding.NewObjectID(25),
			PrescriptionStatus: "eRxSent",
		},
	}

	if treatment1.Equals(treatment3) {
		t.Fatal("treatment1 and treatment3 expected not to be equal")
	}

	treatment4 := &Treatment{
		ID: encoding.NewObjectID(5),
		DrugDBIDs: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName: "Testing",
		DrugName:         "Testing",
		DosageStrength:   "50mg",
		DispenseValue:    12,
		DispenseUnitID:   encoding.NewObjectID(12),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		}, SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		}, ERx: &ERxData{
			PrescriptionID:     encoding.NewObjectID(20),
			ErxMedicationID:    encoding.NewObjectID(26),
			PrescriptionStatus: "eRxSent",
		},
	}

	if !treatment1.Equals(treatment4) {
		t.Fatal("treatment1 and treatment4 expected to be equal")
	}

	treatment5 := &Treatment{
		ID: encoding.NewObjectID(5),
		DrugDBIDs: map[string]string{
			"lexi_gen_product_id":  "123456",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName: "Testing",
		DrugName:         "Testing",
		DosageStrength:   "50mg",
		DispenseValue:    12,
		DispenseUnitID:   encoding.NewObjectID(12),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		}, SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		}, ERx: &ERxData{
			PrescriptionID:     encoding.NewObjectID(20),
			ErxMedicationID:    encoding.NewObjectID(25),
			PrescriptionStatus: "eRxSent",
		},
	}

	if treatment1.Equals(treatment5) {
		t.Fatal("treatment1 and treatment5 expected not to be equal")
	}

	treatment6 := &Treatment{
		ID: encoding.NewObjectID(5),
		DrugDBIDs: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName: "DifferentName",
		DrugName:         "Testing",
		DosageStrength:   "50mg",
		DispenseValue:    12,
		DispenseUnitID:   encoding.NewObjectID(12),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		}, SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		}, ERx: &ERxData{
			PrescriptionID:     encoding.NewObjectID(20),
			ErxMedicationID:    encoding.NewObjectID(25),
			PrescriptionStatus: "eRxSent",
		},
	}

	if !treatment1.Equals(treatment6) {
		t.Fatal("treatment1 and treatment6 expected to be equal")
	}

	treatment7 := &Treatment{
		ID: encoding.NewObjectID(5),
		DrugDBIDs: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName: "DifferentName",
		DrugName:         "Testing",
		DosageStrength:   "50mgs",
		DispenseValue:    12,
		DispenseUnitID:   encoding.NewObjectID(12),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		}, SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		}, ERx: &ERxData{
			PrescriptionID:     encoding.NewObjectID(20),
			ErxMedicationID:    encoding.NewObjectID(25),
			PrescriptionStatus: "eRxSent",
		},
	}

	if treatment1.Equals(treatment7) {
		t.Fatal("treatment1 and treatment6 expected not to be equal")
	}

	treatment8 := &Treatment{}
	if treatment1.Equals(treatment8) {
		t.Fatal("treatment1 and treatment8 expected not to be equal")
	}

	var treatment9, treatment10 *Treatment
	if treatment9.Equals(treatment10) {
		t.Fatal("Null treatments (tretment 9 and 10) expected not to be equal")
	}

	if treatment10.Equals(treatment8) {
		t.Fatal("If one of the treatments is null, then equality should be false")
	}

}

func TestTreatmentEquals_NoErxData(t *testing.T) {
	treatment1 := &Treatment{
		ID: encoding.NewObjectID(5),
		DrugDBIDs: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName: "Testing",
		DrugName:         "Testing",
		DosageStrength:   "50mg",
		DispenseValue:    12,
		DispenseUnitID:   encoding.NewObjectID(12),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		}, SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
	}

	treatment2 := &Treatment{
		ID: encoding.NewObjectID(5),
		DrugDBIDs: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName: "Testing",
		DrugName:         "Testing",
		DosageStrength:   "50mg",
		DispenseValue:    12,
		DispenseUnitID:   encoding.NewObjectID(12),
		NumberRefills: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 1,
		}, SubstitutionsAllowed: false,
		DaysSupply: encoding.NullInt64{
			IsValid:    true,
			Int64Value: 5,
		},
	}

	// treatment1 and treatment 2 should be equal
	if !treatment1.Equals(treatment2) {
		t.Fatal("treatment 1 and 2 expected to be equal")
	}
}

func TestTreatmentListEquals(t *testing.T) {
	treatmentList1 := &TreatmentList{
		Treatments: []*Treatment{
			{
				ID: encoding.NewObjectID(5),
				DrugDBIDs: map[string]string{
					"lexi_gen_product_id":  "12345",
					"lexi_drug_syn_id":     "56789",
					"lexi_synonym_type_id": "123415",
				},
				DrugInternalName: "Testing",
				DrugName:         "Testing",
				DosageStrength:   "50mg",
				DispenseValue:    12,
				DispenseUnitID:   encoding.NewObjectID(12),
				NumberRefills: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 1,
				}, SubstitutionsAllowed: false,
				DaysSupply: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 5,
				},
			},
			{
				ID: encoding.NewObjectID(5),
				DrugDBIDs: map[string]string{
					"lexi_gen_product_id":  "12345",
					"lexi_drug_syn_id":     "56789",
					"lexi_synonym_type_id": "123415",
				},
				DrugInternalName: "DifferentName",
				DrugName:         "Testing",
				DosageStrength:   "50mgs",
				DispenseValue:    12,
				DispenseUnitID:   encoding.NewObjectID(12),
				NumberRefills: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 1,
				}, SubstitutionsAllowed: false,
				DaysSupply: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 5,
				}, ERx: &ERxData{
					PrescriptionID:     encoding.NewObjectID(20),
					ErxMedicationID:    encoding.NewObjectID(25),
					PrescriptionStatus: "eRxSent",
				},
			},
		},
	}

	treatmentList2 := &TreatmentList{
		Treatments: []*Treatment{
			{
				ID: encoding.NewObjectID(5),
				DrugDBIDs: map[string]string{
					"lexi_gen_product_id":  "12345",
					"lexi_drug_syn_id":     "56789",
					"lexi_synonym_type_id": "123415",
				},
				DrugInternalName: "Testing",
				DrugName:         "Testing",
				DosageStrength:   "50mg",
				DispenseValue:    12,
				DispenseUnitID:   encoding.NewObjectID(12),
				NumberRefills: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 1,
				}, SubstitutionsAllowed: false,
				DaysSupply: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 5,
				},
			},
			{
				ID: encoding.NewObjectID(5),
				DrugDBIDs: map[string]string{
					"lexi_gen_product_id":  "12345",
					"lexi_drug_syn_id":     "56789",
					"lexi_synonym_type_id": "123415",
				},
				DrugInternalName: "DifferentName",
				DrugName:         "Testing",
				DosageStrength:   "50mgs",
				DispenseValue:    12,
				DispenseUnitID:   encoding.NewObjectID(12),
				NumberRefills: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 1,
				}, SubstitutionsAllowed: false,
				DaysSupply: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 5,
				}, ERx: &ERxData{
					PrescriptionID:     encoding.NewObjectID(20),
					ErxMedicationID:    encoding.NewObjectID(25),
					PrescriptionStatus: "eRxSent",
				},
			},
		},
	}

	if !treatmentList1.Equals(treatmentList2) {
		t.Fatalf("TreatmentLists expected to be equal")
	}
}

func TestTreatmentListEquals_DifferentLength(t *testing.T) {
	treatmentList1 := &TreatmentList{
		Treatments: []*Treatment{
			{
				ID: encoding.NewObjectID(5),
				DrugDBIDs: map[string]string{
					"lexi_gen_product_id":  "12345",
					"lexi_drug_syn_id":     "56789",
					"lexi_synonym_type_id": "123415",
				},
				DrugInternalName: "Testing",
				DrugName:         "Testing",
				DosageStrength:   "50mg",
				DispenseValue:    12,
				DispenseUnitID:   encoding.NewObjectID(12),
				NumberRefills: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 1,
				}, SubstitutionsAllowed: false,
				DaysSupply: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 5,
				},
			},
		},
	}

	treatmentList2 := &TreatmentList{
		Treatments: []*Treatment{
			{
				ID: encoding.NewObjectID(5),
				DrugDBIDs: map[string]string{
					"lexi_gen_product_id":  "12345",
					"lexi_drug_syn_id":     "56789",
					"lexi_synonym_type_id": "123415",
				},
				DrugInternalName: "Testing",
				DrugName:         "Testing",
				DosageStrength:   "50mg",
				DispenseValue:    12,
				DispenseUnitID:   encoding.NewObjectID(12),
				NumberRefills: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 1,
				}, SubstitutionsAllowed: false,
				DaysSupply: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 5,
				},
			},
			{
				ID: encoding.NewObjectID(5),
				DrugDBIDs: map[string]string{
					"lexi_gen_product_id":  "12345",
					"lexi_drug_syn_id":     "56789",
					"lexi_synonym_type_id": "123415",
				},
				DrugInternalName: "DifferentName",
				DrugName:         "Testing",
				DosageStrength:   "50mgs",
				DispenseValue:    12,
				DispenseUnitID:   encoding.NewObjectID(12),
				NumberRefills: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 1,
				}, SubstitutionsAllowed: false,
				DaysSupply: encoding.NullInt64{
					IsValid:    true,
					Int64Value: 5,
				}, ERx: &ERxData{
					PrescriptionID:     encoding.NewObjectID(20),
					ErxMedicationID:    encoding.NewObjectID(25),
					PrescriptionStatus: "eRxSent",
				},
			},
		},
	}

	if treatmentList1.Equals(treatmentList2) {
		t.Fatalf("TreatmentLists expected to be unequal")
	}
}

func TestTreatmentListEquals_EmptyLists(t *testing.T) {
	treatmentList1 := &TreatmentList{}
	treatmentList2 := &TreatmentList{}
	if !treatmentList2.Equals(treatmentList1) {
		t.Fatalf("TreatmentLists expected to be equal")
	}
}
