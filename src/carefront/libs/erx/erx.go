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
	GetDrugNamesForDoctor(clinicianId int64, prefix string) ([]string, error)
	GetDrugNamesForPatient(clinicianId int64, prefix string) ([]string, error)
	SearchForMedicationStrength(clinicianId int64, medicationName string) ([]string, error)
	SelectMedication(clinicianId int64, medicationName, medicationStrength string) (medication *Medication, err error)
	StartPrescribingPatient(clinicianId int64, patient *common.Patient, treatments []*common.Treatment) error
	SendMultiplePrescriptions(clinicianId int64, patient *common.Patient, treatments []*common.Treatment) ([]int64, error)
	SearchForPharmacies(clinicianId int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error)
	GetPrescriptionStatus(clinicianId, prescriptionId int64) ([]*PrescriptionLog, error)
	GetMedicationList(clinicianId, patientId int64) ([]*Medication, error)
	GetTransmissionErrorDetails(clinicianId int64) ([]*Medication, error)
	GetTransmissionErrorRefillRequestsCount(clinicianId int64) (refillRequests int64, transactionErrors int64, err error)
	IgnoreAlert(clinicianId int64, prescriptionId int64) error
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
	IsControlledSubstance   bool
}

type PrescriptionLog struct {
	PrescriptionStatus string
	AdditionalInfo     string
	LogTimeStamp       time.Time
}
