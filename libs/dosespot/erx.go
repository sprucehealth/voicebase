package dosespot

const (
	PharmacyTypeTwentyFourHours = "TwentyFourHourPharmacy"
	PharmacyTypeMailOrder       = "MailOrder"
	PharmacyTypeLongTermCare    = "LongTermCarePharmacy"
	PharmacyTypeRetail          = "Retail"
	PharmacyTypeSpecialty       = "SpecialtyPharmacy"
)

type API interface {
	ApproveRefillRequest(clinicianID, erxRefillRequestQueueItemID, approvedRefillAmount int64, comments string) (int64, error)
	DenyRefillRequest(clinicianID, erxRefillRequestQueueItemID int64, denialReason, comments string) (int64, error)
	GetDrugNamesForDoctor(clinicianID int64, prefix string) ([]string, error)
	GetDrugNamesForPatient(prefix string) ([]string, error)
	GetPatientDetails(erxPatientID int64) (*PatientUpdate, error)
	GetPharmacyDetails(pharmacyID int64) (*Pharmacy, error)
	GetPrescriptionStatus(clinicianID, prescriptionID int64) ([]*PrescriptionLogInfo, error)
	GetRefillRequestQueueForClinic(clinicianID int64) ([]*RefillRequestQueueItem, error)
	GetTransmissionErrorDetails(clinicianID int64) ([]*TransmissionErrorDetails, error)
	GetTransmissionErrorRefillRequestsCount(clinicianID int64) (refillRequests int64, transactionErrors int64, err error)
	IgnoreAlert(clinicianID int64, prescriptionID int64) error
	SearchForAllergyRelatedMedications(searchTerm string) ([]string, error)
	SearchForMedicationStrength(clinicianID int64, medicationName string) ([]string, error)
	SearchForPharmacies(clinicianID int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*Pharmacy, error)
	SelectMedication(clinicianID int64, medicationName, medicationStrength string) (*MedicationSelectResponse, error)
	SendMultiplePrescriptions(clinicianID, eRxPatientID int64, prescriptionIDs []int64) ([]*SendPrescriptionResult, error)
	StartPrescribingPatient(clinicianID int64, patient *Patient, prescriptions []*Prescription, pharmacySourceID int64) ([]*PatientUpdate, error)
	UpdatePatientInformation(clinicianID int64, patient *Patient, pharmacyID int64) ([]*PatientUpdate, error)
}
