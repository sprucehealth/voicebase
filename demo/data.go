package demo

import (
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"

	"github.com/sprucehealth/backend/pharmacy"
)

func prepareSurescriptsPatients() []*common.Patient {
	patients := make([]*common.Patient, 8)

	patients[0] = &common.Patient{
		FirstName: "Ci",
		LastName:  "Li",
		Gender:    "Male",
		DOB: encoding.DOB{
			Year:  1923,
			Month: 10,
			Day:   18,
		},
		ZipCode: "94115",
		PhoneNumbers: []*common.PhoneNumber{&common.PhoneNumber{
			Phone: "2068773590",
			Type:  "Home",
		},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceId:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		},
		PatientAddress: &common.Address{
			AddressLine1: "12345 Main Street",
			AddressLine2: "Apt 1112",
			City:         "San Francisco",
			State:        "California",
			ZipCode:      "94115",
		},
	}

	patients[1] = &common.Patient{
		Prefix:    "Mr",
		FirstName: "Howard",
		LastName:  "Plower",
		Gender:    "Male",
		DOB: encoding.DOB{
			Year:  1923,
			Month: 10,
			Day:   18,
		},
		ZipCode: "19102",
		PhoneNumbers: []*common.PhoneNumber{
			&common.PhoneNumber{
				Phone: "215-988-6723",
				Type:  "Home",
			},
			&common.PhoneNumber{
				Phone: "4137762738",
				Type:  "Cell",
			},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceId:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		},
		PatientAddress: &common.Address{
			AddressLine1: "76 Deerlake Road",
			City:         "Philadelphia",
			State:        "Pennsylvania",
			ZipCode:      "19102",
		},
	}

	patients[2] = &common.Patient{
		FirstName: "Kara",
		LastName:  "Whiteside",
		Gender:    "female",
		DOB: encoding.DOB{
			Year:  1952,
			Month: 10,
			Day:   11,
		},
		ZipCode: "44306",
		PhoneNumbers: []*common.PhoneNumber{
			&common.PhoneNumber{
				Phone: "3305547754",
				Type:  "Home",
			},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceId:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		},
		PatientAddress: &common.Address{
			AddressLine1: "23230 Seaport",
			City:         "Akron",
			State:        "Ohio",
			ZipCode:      "44306",
		},
	}

	patients[3] = &common.Patient{
		Prefix:    "Ms",
		FirstName: "Debra",
		LastName:  "Tucker",
		Gender:    "female",
		DOB: encoding.DOB{
			Year:  1970,
			Month: 11,
			Day:   01,
		},
		ZipCode: "44103",
		PhoneNumbers: []*common.PhoneNumber{
			&common.PhoneNumber{
				Phone: "4408450398",
				Type:  "Home",
			},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceId:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		},
		PatientAddress: &common.Address{
			AddressLine1: "8331 Everwood Dr.",
			AddressLine2: "Apt 342",
			City:         "Cleveland",
			State:        "Ohio",
			ZipCode:      "44103",
		},
	}

	patients[4] = &common.Patient{
		Prefix:     "Ms",
		FirstName:  "Felicia",
		LastName:   "Flounders",
		MiddleName: "Ann",
		Gender:     "female",
		DOB: encoding.DOB{
			Year:  1980,
			Month: 11,
			Day:   01,
		},
		ZipCode: "20187",
		PhoneNumbers: []*common.PhoneNumber{
			&common.PhoneNumber{
				Phone: "3108620035x2345",
				Type:  "Home",
			},
			&common.PhoneNumber{
				Phone: "3019289283",
				Type:  "Cell",
			},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceId:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		},
		PatientAddress: &common.Address{
			AddressLine1: "6715 Swanson Ave",
			AddressLine2: "Apt 102",
			City:         "Bethesda",
			State:        "Maryland",
			ZipCode:      "20187",
		},
	}

	patients[5] = &common.Patient{
		FirstName:  "Douglas",
		LastName:   "Richardson",
		MiddleName: "R",
		Gender:     "Male",
		DOB: encoding.DOB{
			Year:  1968,
			Month: 9,
			Day:   1,
		},
		ZipCode: "01040",
		PhoneNumbers: []*common.PhoneNumber{
			&common.PhoneNumber{
				Phone: "4137760938",
				Type:  "Home",
			},
			&common.PhoneNumber{
				Phone: "4137762738",
				Type:  "Cell",
			},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceId:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		},
		PatientAddress: &common.Address{
			AddressLine1: "2556 Lane Rd",
			AddressLine2: "Apt 101",
			City:         "Smittyville",
			State:        "Virginia",
			ZipCode:      "01040-2239",
		},
	}

	patients[6] = &common.Patient{
		FirstName: "David",
		LastName:  "Thrower",
		Gender:    "Male",
		DOB: encoding.DOB{
			Year:  1933,
			Month: 2,
			Day:   22,
		},
		ZipCode: "34737",
		PhoneNumbers: []*common.PhoneNumber{
			&common.PhoneNumber{
				Phone: "3526685547",
				Type:  "Home",
			},
			&common.PhoneNumber{
				Phone: "4137762738",
				Type:  "Cell",
			},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceId:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		},
		PatientAddress: &common.Address{
			AddressLine1: "64 Violet Lane",
			AddressLine2: "Apt 101",
			City:         "Howey In The Hills",
			State:        "Florida",
			ZipCode:      "34737",
		},
	}

	patients[7] = &common.Patient{
		Prefix:     "Patient II",
		FirstName:  "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
		LastName:   "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
		MiddleName: "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
		Suffix:     "Junior iii",
		Gender:     "Male",
		DOB: encoding.DOB{
			Year:  1948,
			Month: 1,
			Day:   1,
		},
		ZipCode: "34737",
		PhoneNumbers: []*common.PhoneNumber{
			&common.PhoneNumber{
				Phone: "5719212122x1234567890444",
				Type:  "Home",
			},
			&common.PhoneNumber{
				Phone: "7034445523x4473",
				Type:  "Cell",
			},
			&common.PhoneNumber{
				Phone: "7034445524x4474",
				Type:  "Work",
			},
			&common.PhoneNumber{
				Phone: "7034445522x4472",
				Type:  "Work",
			},
			&common.PhoneNumber{
				Phone: "7034445526x4476",
				Type:  "Home",
			},
		},
		Pharmacy: &pharmacy.PharmacyData{
			SourceId:     47731,
			AddressLine1: "116 New Montgomery St",
			Name:         "CA pharmacy store 10.6",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "92804",
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
		},
		PatientAddress: &common.Address{
			AddressLine1: "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
			AddressLine2: "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
			City:         "!\"#$%'+,-/:;=?@[\\]^_`{|}~0000&",
			State:        "Colorado",
			ZipCode:      "94115",
		},
	}
	return patients
}

func prepareDemoPatients(n int64) []*common.Patient {
	patients := make([]*common.Patient, n)
	for i := int64(0); i < n; i++ {
		patients[i] = &common.Patient{
			FirstName: "Kunal",
			LastName:  "Jham",
			Gender:    "male",
			DOB: encoding.DOB{
				Year:  1987,
				Month: 11,
				Day:   8,
			},
			ZipCode: "94115",
			PhoneNumbers: []*common.PhoneNumber{&common.PhoneNumber{
				Phone: "2068773590",
				Type:  "Home",
			},
			},
			Pharmacy: &pharmacy.PharmacyData{
				SourceId:     47731,
				AddressLine1: "116 New Montgomery St",
				Name:         "CA pharmacy store 10.6",
				City:         "San Francisco",
				State:        "CA",
				Postal:       "92804",
				Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			},
			PatientAddress: &common.Address{
				AddressLine1: "12345 Main Street",
				AddressLine2: "Apt 1112",
				City:         "San Francisco",
				State:        "California",
				ZipCode:      "94115",
			},
		}
	}
	return patients
}

var favoriteTreatmentPlans = map[string]*common.FavoriteTreatmentPlan{
	"doxy_and_tretinoin": &common.FavoriteTreatmentPlan{
		Name: "Doxy and Tretinoin",
		TreatmentList: &common.TreatmentList{
			Treatments: []*common.Treatment{
				&common.Treatment{
					DrugDBIds: map[string]string{
						"ndc": "00245904519",
						"lexi_gen_product_id":  "3162",
						"lexi_synonym_type_id": "59",
						"lexi_drug_syn_id":     "19573",
					},
					DrugInternalName: "Tretinoin Topical (topical - cream)",
					DrugName:         "Tretinoin Topical",
					DrugRoute:        "topical",
					DrugForm:         "cream",
					DosageStrength:   "0.025%",
					DispenseValue:    encoding.HighPrecisionFloat64(1.0000000000),
					DispenseUnitId:   encoding.NewObjectId(29),
					NumberRefills: encoding.NullInt64{
						IsValid:    true,
						Int64Value: 2,
					},
					SubstitutionsAllowed: true,
					PharmacyNotes:        "For the treatment of acne vulgaris (706.0)",
					PatientInstructions:  "Apply pea-sized amount over affected area at night. Start every other night for 2-4 weeks and gradually increase id tolerated to every night",
				},
				&common.Treatment{
					DrugDBIds: map[string]string{
						"ndc": "00003081240",
						"lexi_gen_product_id":  "1161",
						"lexi_synonym_type_id": "59",
						"lexi_drug_syn_id":     "23011",
					},
					DrugInternalName: "Doxycycline (oral - tablet)",
					DrugName:         "Doxycycline",
					DrugRoute:        "oral",
					DrugForm:         "tablet",
					DosageStrength:   "hyclate 100 mg",
					DispenseValue:    encoding.HighPrecisionFloat64(180.0000000000),
					DispenseUnitId:   encoding.NewObjectId(26),
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
					State: common.STATE_ADDED,
				},
				{
					Text:  "Apply a lightweight moisturizer with SPF 50.",
					State: common.STATE_ADDED,
				},
				{
					Text:  "Take doxycycline 100mg with breakfast.",
					State: common.STATE_ADDED,
				},
				{
					Text:  "Take doxycycline 100mg with dinner.",
					State: common.STATE_ADDED,
				},
				{
					Text:  "Dry your face completely.",
					State: common.STATE_ADDED,
				},
				{
					Text:  "Apply pea-sized amount of tretinoin cream to entire face.",
					State: common.STATE_ADDED,
				},
				{
					Text:  "Apply pea-size amount of benzoyl peroxide cream to entire face.",
					State: common.STATE_ADDED,
				},
				{
					Text:  "Apply nighttime moisturizer as needed.",
					State: common.STATE_ADDED,
				},
			},
			Sections: []*common.RegimenSection{
				{
					Name: "Morning",
					Steps: []*common.DoctorInstructionItem{
						{
							Text:  "Wash your face with a gentle cleanser",
							State: common.STATE_ADDED,
						},
						{
							Text:  "Apply a lightweight moisturizer with SPF 50.",
							State: common.STATE_ADDED,
						},
						{
							Text:  "Take doxycycline 100mg with breakfast.",
							State: common.STATE_ADDED,
						},
					},
				},
				{
					Name: "Nighttime",
					Steps: []*common.DoctorInstructionItem{
						{
							Text:  "Take doxycycline 100mg with dinner.",
							State: common.STATE_ADDED,
						},
						{
							Text:  "Wash your face with a gentle cleanser",
							State: common.STATE_ADDED,
						},
						{
							Text:  "Dry your face completely.",
							State: common.STATE_ADDED,
						},
						{
							Text:  "Apply pea-sized amount of tretinoin cream to entire face.",
							State: common.STATE_ADDED,
						},
						{
							Text:  "Apply nighttime moisturizer as needed.",
							State: common.STATE_ADDED,
						},
					},
				},
			},
		},
	},
}

var messageForTreatmentPlan = `Dear %s,

I've taken a look at your pictures, and from what I can tell, you have moderate inflammatory and comedonal acne. I've put together a treatment regimen for you that will take roughly 3 months to take full effect. Please stick with it as best as you can, unless you are having a concerning complication. Often times, acne gets slightly worse before it gets better.

Please keep in mind finding the right "recipe" to treat your acne may take some tweaking. As always, feel free to communicate any questions or issues you have along the way.  

Sincerely,
Dr. %s
`
