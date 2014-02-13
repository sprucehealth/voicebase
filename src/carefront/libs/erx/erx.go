package erx

import (
	"carefront/common"
	pharmacySearch "carefront/libs/pharmacy"
	"time"
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
	StartPrescribingPatient(patient *common.Patient, treatments []*common.Treatment) error
	SendMultiplePrescriptions(patient *common.Patient, treatments []*common.Treatment) ([]int64, error)
	SearchForPharmacies(city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error)
	GetPrescriptionStatus(prescriptionId int64) ([]*PrescriptionLog, error)
	GetMedicationList(patientId int64) ([]*Medication, error)
	GetTransmissionErrorDetails() ([]*Medication, error)
	GetTransmissionErrorRefillRequestsCount() (refillRequests int64, transactionErrors int64, err error)
}

type Medication struct {
	ErxMedicationId         int64
	DoseSpotPrescriptionId  int64
	PrescriptionStatus      string
	PrescriptionDate        *time.Time
	DrugDBIds               map[string]string
	OTC                     bool
	DispenseUnitId          int64
	DispenseUnitDescription string
	ErrorTimeStamp          *time.Time
	ErrorDetails            string
	DisplayName             string
	Strength                string
	Refills                 int64
	DaysSupply              int64
	Dispense                string
	Instructions            string
	PharmacyId              int64
	PharmacyNotes           string
	NoSubstitutions         bool
	RxReferenceNumber       string
}

type PrescriptionLog struct {
	PrescriptionStatus string
	LogTimeStamp       time.Time
}
