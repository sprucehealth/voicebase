package common

import "testing"

func TestTreatmentEquals(t *testing.T) {

	treatment1 := &Treatment{
		Id: NewObjectId(5),
		DrugDBIds: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName:     "Testing",
		DrugName:             "Testing",
		DosageStrength:       "50mg",
		DispenseValue:        12,
		DispenseUnitId:       NewObjectId(12),
		NumberRefills:        1,
		SubstitutionsAllowed: false,
		DaysSupply:           5,
		ERx: &ERxData{
			PrescriptionId:     NewObjectId(20),
			ErxMedicationId:    NewObjectId(25),
			PrescriptionStatus: "eRxSent",
		},
	}

	treatment2 := &Treatment{
		Id: NewObjectId(5),
		DrugDBIds: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName:     "Testing",
		DrugName:             "Testing",
		DosageStrength:       "50mg",
		DispenseValue:        12,
		DispenseUnitId:       NewObjectId(12),
		NumberRefills:        1,
		SubstitutionsAllowed: false,
		DaysSupply:           5,
		ERx: &ERxData{
			PrescriptionId:     NewObjectId(20),
			ErxMedicationId:    NewObjectId(25),
			PrescriptionStatus: "eRxSent",
		},
	}

	// treatment1 and treatment 2 should be equal
	if !treatment1.Equals(treatment2) {
		t.Fatal("treatment 1 and 2 expected to be equal")
	}

	treatment3 := &Treatment{
		Id: NewObjectId(5),
		DrugDBIds: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName:     "Testing",
		DrugName:             "Testing",
		DosageStrength:       "50mg",
		DispenseValue:        12,
		DispenseUnitId:       NewObjectId(12),
		NumberRefills:        1,
		SubstitutionsAllowed: false,
		DaysSupply:           5,
		ERx: &ERxData{
			PrescriptionId:     NewObjectId(21),
			ErxMedicationId:    NewObjectId(25),
			PrescriptionStatus: "eRxSent",
		},
	}

	if treatment1.Equals(treatment3) {
		t.Fatal("treatment1 and treatment3 expected not to be equal")
	}

	treatment4 := &Treatment{
		Id: NewObjectId(5),
		DrugDBIds: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName:     "Testing",
		DrugName:             "Testing",
		DosageStrength:       "50mg",
		DispenseValue:        12,
		DispenseUnitId:       NewObjectId(12),
		NumberRefills:        1,
		SubstitutionsAllowed: false,
		DaysSupply:           5,
		ERx: &ERxData{
			PrescriptionId:     NewObjectId(20),
			ErxMedicationId:    NewObjectId(26),
			PrescriptionStatus: "eRxSent",
		},
	}

	if !treatment1.Equals(treatment4) {
		t.Fatal("treatment1 and treatment4 expected to be equal")
	}

	treatment5 := &Treatment{
		Id: NewObjectId(5),
		DrugDBIds: map[string]string{
			"lexi_gen_product_id":  "123456",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName:     "Testing",
		DrugName:             "Testing",
		DosageStrength:       "50mg",
		DispenseValue:        12,
		DispenseUnitId:       NewObjectId(12),
		NumberRefills:        1,
		SubstitutionsAllowed: false,
		DaysSupply:           5,
		ERx: &ERxData{
			PrescriptionId:     NewObjectId(20),
			ErxMedicationId:    NewObjectId(25),
			PrescriptionStatus: "eRxSent",
		},
	}

	if treatment1.Equals(treatment5) {
		t.Fatal("treatment1 and treatment5 expected not to be equal")
	}

	treatment6 := &Treatment{
		Id: NewObjectId(5),
		DrugDBIds: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName:     "DifferentName",
		DrugName:             "Testing",
		DosageStrength:       "50mg",
		DispenseValue:        12,
		DispenseUnitId:       NewObjectId(12),
		NumberRefills:        1,
		SubstitutionsAllowed: false,
		DaysSupply:           5,
		ERx: &ERxData{
			PrescriptionId:     NewObjectId(20),
			ErxMedicationId:    NewObjectId(25),
			PrescriptionStatus: "eRxSent",
		},
	}

	if !treatment1.Equals(treatment6) {
		t.Fatal("treatment1 and treatment6 expected to be equal")
	}

	treatment7 := &Treatment{
		Id: NewObjectId(5),
		DrugDBIds: map[string]string{
			"lexi_gen_product_id":  "12345",
			"lexi_drug_syn_id":     "56789",
			"lexi_synonym_type_id": "123415",
		},
		DrugInternalName:     "DifferentName",
		DrugName:             "Testing",
		DosageStrength:       "50mgs",
		DispenseValue:        12,
		DispenseUnitId:       NewObjectId(12),
		NumberRefills:        1,
		SubstitutionsAllowed: false,
		DaysSupply:           5,
		ERx: &ERxData{
			PrescriptionId:     NewObjectId(20),
			ErxMedicationId:    NewObjectId(25),
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
