package erx

import (
	"time"

	"github.com/sprucehealth/backend/common"
	pharmacySearch "github.com/sprucehealth/backend/pharmacy"
)

const (
	PHARMACY_TYPE_TWENTY_FOUR_HOURS = "TwentyFourHourPharmacy"
	PHARMACY_TYPE_MAIL_ORDER        = "MailOrder"
	PHARMACY_TYPE_LONG_TERM_CARE    = "LongTermCarePharmacy"
	PHARMACY_TYPE_RETAIL            = "Retail"
	PHARMACY_TYPE_SPECIALTY         = "SpecialtyPharmacy"
)

type ERxAPI interface {
	GetDrugNamesForDoctor(clinicianID int64, prefix string) ([]string, error)
	GetDrugNamesForPatient(prefix string) ([]string, error)
	SearchForAllergyRelatedMedications(searchTerm string) ([]string, error)
	SearchForMedicationStrength(clinicianID int64, medicationName string) ([]string, error)
	SelectMedication(clinicianID int64, medicationName, medicationStrength string) (medication *common.Treatment, err error)
	UpdatePatientInformation(clinicianID int64, patient *common.Patient) error
	StartPrescribingPatient(clinicianID int64, patient *common.Patient, treatments []*common.Treatment, pharmacySourceId int64) error
	SendMultiplePrescriptions(clinicianID int64, patient *common.Patient, treatments []*common.Treatment) ([]*common.Treatment, error)
	SearchForPharmacies(clinicianID int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error)
	GetPrescriptionStatus(clinicianID, prescriptionID int64) ([]*PrescriptionLog, error)
	GetTransmissionErrorDetails(clinicianID int64) ([]*common.Treatment, error)
	GetTransmissionErrorRefillRequestsCount(clinicianID int64) (refillRequests int64, transactionErrors int64, err error)
	IgnoreAlert(clinicianID int64, prescriptionID int64) error
	GetRefillRequestQueueForClinic(clinicianID int64) ([]*common.RefillRequestItem, error)
	GetPatientDetails(erxPatientID int64) (*common.Patient, error)
	GetPharmacyDetails(pharmacyID int64) (*pharmacySearch.PharmacyData, error)
	ApproveRefillRequest(clinicianID, erxRefillRequestQueueItemId, approvedRefillAmount int64, comments string) (int64, error)
	DenyRefillRequest(clinicianID, erxRefillRequestQueueItemId int64, denialReason string, comments string) (int64, error)
}

type PrescriptionLog struct {
	PrescriptionStatus string
	AdditionalInfo     string
	LogTimestamp       time.Time
}
