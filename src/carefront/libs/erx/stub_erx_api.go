package erx

import (
	"carefront/common"
	pharmacySearch "carefront/libs/pharmacy"
	"fmt"
	"time"
)

type StubErxService struct {
	PatientErxId                         int64
	RefillRequestPrescriptionIds         map[int64]int64
	PatientDetailsToReturn               *common.Patient
	PharmacyDetailsToReturn              *pharmacySearch.PharmacyData
	RefillRxRequestQueueToReturn         []*common.RefillRequestItem
	PrescriptionIdsToReturn              []int64
	PrescriptionIdToPrescriptionStatuses map[int64][]common.StatusEvent
	SelectedMedicationToReturn           *common.Treatment
}

func (s *StubErxService) GetDrugNamesForDoctor(clinicianId int64, prefix string) ([]string, error) {
	return nil, nil
}

func (s *StubErxService) GetDrugNamesForPatient(clinicianId int64, prefix string) ([]string, error) {
	return nil, nil
}

func (s *StubErxService) SearchForMedicationStrength(clinicianId int64, medicationName string) ([]string, error) {
	return nil, nil
}

func (s *StubErxService) SelectMedication(clinicianId int64, medicationName, medicationStrength string) (medication *common.Treatment, err error) {
	return s.SelectedMedicationToReturn, nil
}

func (s *StubErxService) StartPrescribingPatient(clinicianId int64, Patient *common.Patient, Treatments []*common.Treatment) error {
	fmt.Println("Starting to prescribe patient")
	// walk through the treatments and assign them each a prescription id
	// assumption here is that there are as many prescription ids to return as there are treatments
	Patient.ERxPatientId = common.NewObjectId(s.PatientErxId)
	for i, treatment := range Treatments {
		if treatment.ERx == nil {
			treatment.ERx = &common.ERxData{}
		}
		treatment.ERx.PrescriptionId = common.NewObjectId(s.PrescriptionIdsToReturn[i])
	}
	return nil
}

func (s *StubErxService) SendMultiplePrescriptions(clinicianId int64, Patient *common.Patient, Treatments []*common.Treatment) ([]int64, error) {
	// nothing to do here given that the act of sending a prescription successfully does not change the state of the system
	fmt.Println("Sending multiple prescriptions")
	return nil, nil
}

func (s *StubErxService) SearchForPharmacies(clinicianId int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error) {
	return nil, nil
}

func (s *StubErxService) GetPrescriptionStatus(clinicianId int64, prescriptionId int64) ([]*PrescriptionLog, error) {
	prescriptionStatuses := s.PrescriptionIdToPrescriptionStatuses[prescriptionId]
	prescriptionLogs := make([]*PrescriptionLog, 0)
	for _, prescriptionStatus := range prescriptionStatuses {
		prescriptionLogs = append(prescriptionLogs, &PrescriptionLog{
			PrescriptionStatus: prescriptionStatus.Status,
			LogTimestamp:       time.Now(),
			AdditionalInfo:     prescriptionStatus.StatusDetails,
		})
	}

	return prescriptionLogs, nil
}

func (s *StubErxService) GetMedicationList(clinicianId int64, PatientId int64) ([]*common.Treatment, error) {
	return nil, nil
}

func (s *StubErxService) GetTransmissionErrorDetails(clinicianId int64) ([]*common.Treatment, error) {
	return nil, nil
}

func (s *StubErxService) GetTransmissionErrorRefillRequestsCount(clinicianId int64) (refillRequests int64, transactionErrors int64, err error) {
	return
}

func (s *StubErxService) IgnoreAlert(clinicianId int64, prescriptionId int64) error {
	return nil
}

func (s *StubErxService) GetRefillRequestQueueForClinic() ([]*common.RefillRequestItem, error) {
	return s.RefillRxRequestQueueToReturn, nil
}

func (s *StubErxService) GetPatientDetails(erxPatientId int64) (*common.Patient, error) {
	return s.PatientDetailsToReturn, nil
}

func (s *StubErxService) GetPharmacyDetails(pharmacyId int64) (*pharmacySearch.PharmacyData, error) {
	return s.PharmacyDetailsToReturn, nil
}

func (s *StubErxService) ApproveRefillRequest(clinicianId, erxRefillRequestQueueItemId, approvedRefillAmount int64, comments string) (int64, error) {
	return s.RefillRequestPrescriptionIds[erxRefillRequestQueueItemId], nil
}

func (s *StubErxService) DenyRefillRequest(clinicianId, erxRefillRequestQueueItemId int64, denialReason string, comments string) (int64, error) {
	return s.RefillRequestPrescriptionIds[erxRefillRequestQueueItemId], nil
}

func (s *StubErxService) UpdatePatientInformation(clinicianId int64, patient *common.Patient) error {
	return nil
}
