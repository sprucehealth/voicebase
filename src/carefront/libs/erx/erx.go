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
	SelectMedication(clinicianId int64, medicationName, medicationStrength string) (medication *common.Treatment, err error)
	UpdatePatientInformation(clinicianId int64, patient *common.Patient) error
	StartPrescribingPatient(clinicianId int64, patient *common.Patient, treatments []*common.Treatment) error
	SendMultiplePrescriptions(clinicianId int64, patient *common.Patient, treatments []*common.Treatment) ([]int64, error)
	SearchForPharmacies(clinicianId int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error)
	GetPrescriptionStatus(clinicianId, prescriptionId int64) ([]*PrescriptionLog, error)
	GetMedicationList(clinicianId, patientId int64) ([]*common.Treatment, error)
	GetTransmissionErrorDetails(clinicianId int64) ([]*common.Treatment, error)
	GetTransmissionErrorRefillRequestsCount(clinicianId int64) (refillRequests int64, transactionErrors int64, err error)
	IgnoreAlert(clinicianId int64, prescriptionId int64) error
	GetRefillRequestQueueForClinic() ([]*common.RefillRequestItem, error)
	GetPatientDetails(erxPatientId int64) (*common.Patient, error)
	GetPharmacyDetails(pharmacyId int64) (*pharmacySearch.PharmacyData, error)
	ApproveRefillRequest(clinicianId, erxRefillRequestQueueItemId, approvedRefillAmount int64, comments string) (int64, error)
	DenyRefillRequest(clinicianId, erxRefillRequestQueueItemId int64, denialReason string, comments string) (int64, error)
}

type PrescriptionLog struct {
	PrescriptionStatus string
	AdditionalInfo     string
	LogTimestamp       time.Time
}
