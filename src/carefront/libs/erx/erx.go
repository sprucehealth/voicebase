package erx

import (
	"carefront/common"
	pharmacySearch "carefront/libs/pharmacy"
)

const (
	PHARMACY_TYPE_TWENTY_FOUR_HOURS = "TwentyFourHourPharmacy"
	PHARMACY_TYPE_MAIL_ORDER        = "MailOrder"
	PHARMACY_TYPE_LONG_TERM_CARE    = "LongTermCarePharmacy"
	PHARMACY_TYPE_RETAIL            = "Retail"
	PHARMACY_TYPE_SPECIALTY         = "SpecialtyPharmacy"
)

type ERxAPI interface {
	GetDrugNamesForDoctor(prefix string) ([]string, error)
	GetDrugNamesForPatient(prefix string) ([]string, error)
	SearchForMedicationStrength(medicationName string) ([]string, error)
	SelectMedication(medicationName, medicationStrength string) (medication *Medication, err error)
	StartPrescribingPatient(Patient *common.Patient, Treatments []*common.Treatment) error
	SendMultiplePrescriptions(Patient *common.Patient, Treatments []*common.Treatment) ([]int64, error)
	SearchForPharmacies(city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error)
}

type Medication struct {
	DrugDBIds               map[string]string
	OTC                     bool
	DispenseUnitId          int
	DispenseUnitDescription string
}
