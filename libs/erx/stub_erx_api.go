package erx

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	pharmacySearch "github.com/sprucehealth/backend/pharmacy"
)

type StubErxService struct {
	PatientErxID int64

	RefillRequestPrescriptionIDs         map[int64]int64
	PatientDetailsToReturn               *common.Patient
	PharmacyDetailsToReturn              *pharmacySearch.PharmacyData
	RefillRxRequestQueueToReturn         []*common.RefillRequestItem
	TransmissionErrorsForPrescriptionIds []int64
	PrescriptionIDsToReturn              []int64
	PrescriptionIDToPrescriptionStatuses map[int64][]common.StatusEvent
	PharmacyToSendPrescriptionTo         int64
	ExpectedRxReferenceNumber            string

	SelectMedicationFunc func(clinicianID int64, name, strength string) (*MedicationSelectResponse, error)
}

func (s *StubErxService) GetDrugNamesForDoctor(clinicianID int64, prefix string) ([]string, error) {
	return nil, nil
}

func (s *StubErxService) GetDrugNamesForPatient(prefix string) ([]string, error) {
	return nil, nil
}

func (s *StubErxService) SearchForAllergyRelatedMedications(searchTerm string) ([]string, error) {
	return nil, nil
}

func (s *StubErxService) SearchForMedicationStrength(clinicianID int64, medicationName string) ([]string, error) {
	return nil, nil
}

func (s *StubErxService) SelectMedication(clinicianID int64, medicationName, medicationStrength string) (*MedicationSelectResponse, error) {
	if s.SelectMedicationFunc != nil {
		return s.SelectMedicationFunc(clinicianID, medicationName, medicationStrength)
	}
	return nil, nil
}

func (s *StubErxService) StartPrescribingPatient(clinicianID int64, Patient *common.Patient, Treatments []*common.Treatment, pharmacySourceID int64) error {
	if s.PharmacyToSendPrescriptionTo != 0 && s.PharmacyToSendPrescriptionTo != pharmacySourceID {
		return fmt.Errorf("Expected to send treatment to pharmacy with sourceId %d instead it was attempted to be sent to pharmacy with id %d", s.PharmacyToSendPrescriptionTo, pharmacySourceID)
	}

	if s.ExpectedRxReferenceNumber != "" && Treatments[0].ERx.ErxReferenceNumber != s.ExpectedRxReferenceNumber {
		return fmt.Errorf("Expected the rx reference number to be %s instead it was %s", s.ExpectedRxReferenceNumber, Treatments[0].ERx.ErxReferenceNumber)
	}

	if len(s.PrescriptionIDsToReturn) == 0 {
		return nil
	}

	// walk through the treatments and assign them each a prescription id
	// assumption here is that there are as many prescription ids to return as there are treatments
	Patient.ERxPatientID = encoding.DeprecatedNewObjectID(s.PatientErxID)
	for i, treatment := range Treatments {
		if treatment.ERx == nil {
			treatment.ERx = &common.ERxData{}
		}
		treatment.ERx.PrescriptionID = encoding.DeprecatedNewObjectID(s.PrescriptionIDsToReturn[i])
	}

	return nil
}

func (s *StubErxService) SendMultiplePrescriptions(clinicianID int64, Patient *common.Patient, Treatments []*common.Treatment) ([]*common.Treatment, error) {
	return nil, nil
}

func (s *StubErxService) SearchForPharmacies(clinicianID int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error) {
	return nil, nil
}

func (s *StubErxService) GetPrescriptionStatus(clinicianID int64, prescriptionID int64) ([]*PrescriptionLog, error) {
	prescriptionStatuses := s.PrescriptionIDToPrescriptionStatuses[prescriptionID]
	prescriptionLogs := make([]*PrescriptionLog, len(prescriptionStatuses))

	for i, prescriptionStatus := range prescriptionStatuses {
		prescriptionLogs[i] = &PrescriptionLog{
			PrescriptionStatus: prescriptionStatus.Status,
			LogTimestamp:       time.Now(),
			AdditionalInfo:     prescriptionStatus.StatusDetails,
		}
	}

	return prescriptionLogs, nil
}

func (s *StubErxService) GetTransmissionErrorDetails(clinicianID int64) ([]*common.Treatment, error) {
	timestamp := time.Now()
	transmissionErrors := make([]*common.Treatment, len(s.TransmissionErrorsForPrescriptionIds))
	for i, prescriptionID := range s.TransmissionErrorsForPrescriptionIds {
		transmissionErrors[i] = &common.Treatment{
			ERx: &common.ERxData{
				PrescriptionID:        encoding.DeprecatedNewObjectID(prescriptionID),
				TransmissionErrorDate: &timestamp,
			},
		}
	}
	return transmissionErrors, nil
}

func (s *StubErxService) GetTransmissionErrorRefillRequestsCount(clinicianID int64) (refillRequests int64, transactionErrors int64, err error) {
	return
}

func (s *StubErxService) IgnoreAlert(clinicianID int64, prescriptionID int64) error {
	return nil
}

func (s *StubErxService) GetRefillRequestQueueForClinic(clincianID int64) ([]*common.RefillRequestItem, error) {
	return s.RefillRxRequestQueueToReturn, nil
}

func (s *StubErxService) GetPatientDetails(erxPatientID int64) (*common.Patient, error) {
	return s.PatientDetailsToReturn, nil
}

func (s *StubErxService) GetPharmacyDetails(pharmacyID int64) (*pharmacySearch.PharmacyData, error) {
	return s.PharmacyDetailsToReturn, nil
}

func (s *StubErxService) ApproveRefillRequest(clinicianID, erxRefillRequestQueueItemID, approvedRefillAmount int64, comments string) (int64, error) {
	return s.RefillRequestPrescriptionIDs[erxRefillRequestQueueItemID], nil
}

func (s *StubErxService) DenyRefillRequest(clinicianID, erxRefillRequestQueueItemID int64, denialReason string, comments string) (int64, error) {
	return s.RefillRequestPrescriptionIDs[erxRefillRequestQueueItemID], nil
}

func (s *StubErxService) UpdatePatientInformation(clinicianID int64, patient *common.Patient) error {
	return nil
}
