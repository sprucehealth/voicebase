package demo

import (
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
)

var favoriteTreatmentPlans = map[string]*common.FavoriteTreatmentPlan{
	"doxy_and_tretinoin": &common.FavoriteTreatmentPlan{
		Name: "Doxy and Tretinoin",
		Note: messageForTreatmentPlan,
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{
				&common.Treatment{
					DrugDBIDs: map[string]string{
						"ndc": "00245904519",
						"lexi_gen_product_id":  "3162",
						"lexi_synonym_type_id": "59",
						"lexi_drug_syn_id":     "19573",
					},
					DrugInternalName: "Tretinoin (topical - cream)",
					DrugName:         "Tretinoin",
					DrugRoute:        "topical",
					DrugForm:         "cream",
					GenericDrugName:  "tretinoin",
					DosageStrength:   "0.025%",
					DispenseValue:    encoding.HighPrecisionFloat64(1.0000000000),
					DispenseUnitID:   encoding.NewObjectID(29),
					NumberRefills: encoding.NullInt64{
						IsValid:    true,
						Int64Value: 2,
					},
					SubstitutionsAllowed: true,
					PharmacyNotes:        "For the treatment of acne vulgaris (706.0)",
					PatientInstructions:  "Apply pea-sized amount over affected area at night. Start every other night for 2-4 weeks and gradually increase id tolerated to every night",
				},
				&common.Treatment{
					DrugDBIDs: map[string]string{
						"ndc": "00003081240",
						"lexi_gen_product_id":  "1161",
						"lexi_synonym_type_id": "59",
						"lexi_drug_syn_id":     "23011",
					},
					DrugInternalName: "Doxycycline (oral - tablet)",
					DrugName:         "Doxycycline",
					DrugRoute:        "oral",
					DrugForm:         "tablet",
					GenericDrugName:  "doxycycline",
					DosageStrength:   "hyclate 100 mg",
					DispenseValue:    encoding.HighPrecisionFloat64(180.0000000000),
					DispenseUnitID:   encoding.NewObjectID(26),
					NumberRefills: encoding.NullInt64{
						IsValid:    true,
						Int64Value: 0,
					},
					SubstitutionsAllowed: true,
					PatientInstructions:  "Take twice daily with small amount of food. Remain upright for 30 minutes after taking.",
				},
			},
		},
		RegimenPlan: &common.RegimenPlan{
			AllSteps: []*common.DoctorInstructionItem{
				{
					Text:  "Wash your face with a gentle cleanser",
					State: common.StateAdded,
				},
				{
					Text:  "Apply a lightweight moisturizer with SPF 50.",
					State: common.StateAdded,
				},
				{
					Text:  "Take doxycycline 100mg with breakfast.",
					State: common.StateAdded,
				},
				{
					Text:  "Take doxycycline 100mg with dinner.",
					State: common.StateAdded,
				},
				{
					Text:  "Dry your face completely.",
					State: common.StateAdded,
				},
				{
					Text:  "Apply pea-sized amount of tretinoin cream to entire face.",
					State: common.StateAdded,
				},
				{
					Text:  "Apply pea-size amount of benzoyl peroxide cream to entire face.",
					State: common.StateAdded,
				},
				{
					Text:  "Apply nighttime moisturizer as needed.",
					State: common.StateAdded,
				},
			},
			Sections: []*common.RegimenSection{
				{
					Name: "Morning",
					Steps: []*common.DoctorInstructionItem{
						{
							Text:  "Wash your face with a gentle cleanser",
							State: common.StateAdded,
						},
						{
							Text:  "Apply a lightweight moisturizer with SPF 50.",
							State: common.StateAdded,
						},
						{
							Text:  "Take doxycycline 100mg with breakfast.",
							State: common.StateAdded,
						},
					},
				},
				{
					Name: "Nighttime",
					Steps: []*common.DoctorInstructionItem{
						{
							Text:  "Take doxycycline 100mg with dinner.",
							State: common.StateAdded,
						},
						{
							Text:  "Wash your face with a gentle cleanser",
							State: common.StateAdded,
						},
						{
							Text:  "Dry your face completely.",
							State: common.StateAdded,
						},
						{
							Text:  "Apply pea-sized amount of tretinoin cream to entire face.",
							State: common.StateAdded,
						},
						{
							Text:  "Apply nighttime moisturizer as needed.",
							State: common.StateAdded,
						},
					},
				},
			},
		},
	},
}

var messageForTreatmentPlan = `I've taken a look at your pictures, and from what I can tell, you have moderate inflammatory and comedonal acne. I've put together a treatment regimen for you that will take roughly 3 months to take full effect. Please stick with it as best as you can, unless you are having a concerning complication. Often times, acne gets slightly worse before it gets better.

Please keep in mind finding the right "recipe" to treat your acne may take some tweaking. As always, feel free to communicate any questions or issues you have along the way.`
