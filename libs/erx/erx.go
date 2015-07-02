package erx

import (
	"time"

	"github.com/sprucehealth/backend/common"
	pharmacySearch "github.com/sprucehealth/backend/pharmacy"
)

const (
	PharmacyTypeTwentyFourHours = "TwentyFourHourPharmacy"
	PharmacyTypeMailOrder       = "MailOrder"
	PharmacyTypeLongTermCare    = "LongTermCarePharmacy"
	PharmacyTypeRetail          = "Retail"
	PharmacyTypeSpecialty       = "SpecialtyPharmacy"
)

type ERxAPI interface {
	ApproveRefillRequest(clinicianID, erxRefillRequestQueueItemID, approvedRefillAmount int64, comments string) (int64, error)
	DenyRefillRequest(clinicianID, erxRefillRequestQueueItemID int64, denialReason, comments string) (int64, error)
	GetDrugNamesForDoctor(clinicianID int64, prefix string) ([]string, error)
	GetDrugNamesForPatient(prefix string) ([]string, error)
	GetPatientDetails(erxPatientID int64) (*common.Patient, error)
	GetPharmacyDetails(pharmacyID int64) (*pharmacySearch.PharmacyData, error)
	GetPrescriptionStatus(clinicianID, prescriptionID int64) ([]*PrescriptionLog, error)
	GetRefillRequestQueueForClinic(clinicianID int64) ([]*common.RefillRequestItem, error)
	GetTransmissionErrorDetails(clinicianID int64) ([]*common.Treatment, error)
	GetTransmissionErrorRefillRequestsCount(clinicianID int64) (refillRequests int64, transactionErrors int64, err error)
	IgnoreAlert(clinicianID int64, prescriptionID int64) error
	SearchForAllergyRelatedMedications(searchTerm string) ([]string, error)
	SearchForMedicationStrength(clinicianID int64, medicationName string) ([]string, error)
	SearchForPharmacies(clinicianID int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error)
	SelectMedication(clinicianID int64, medicationName, medicationStrength string) (*MedicationSelectResponse, error)
	SendMultiplePrescriptions(clinicianID int64, patient *common.Patient, treatments []*common.Treatment) ([]*common.Treatment, error)
	StartPrescribingPatient(clinicianID int64, patient *common.Patient, treatments []*common.Treatment, pharmacySourceID int64) error
	UpdatePatientInformation(clinicianID int64, patient *common.Patient) error
}

type PrescriptionLog struct {
	PrescriptionStatus string
	AdditionalInfo     string
	LogTimestamp       time.Time
}
