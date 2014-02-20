package erx

import (
	"carefront/common"
	pharmacySearch "carefront/libs/pharmacy"
	"fmt"
)

type StubErxService struct {
	PatientErxId                       int64
	PrescriptionIdsToReturn            []int64
	PrescriptionIdToPrescriptionStatus map[int64]string
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
	return nil, nil
}

func (s *StubErxService) StartPrescribingPatient(clinicianId int64, Patient *common.Patient, Treatments []*common.Treatment) error {
	fmt.Println("Starting to prescribe patient")
	// walk through the treatments and assign them each a prescription id
	// assumption here is that there are as many prescription ids to return as there are treatments
	Patient.ERxPatientId = common.NewObjectId(s.PatientErxId)
	for i, treatment := range Treatments {
		treatment.PrescriptionId = common.NewObjectId(s.PrescriptionIdsToReturn[i])
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
	return nil, nil
}

func (s *StubErxService) GetMedicationList(clinicianId int64, PatientId int64) ([]*common.Treatment, error) {
	medications := make([]*common.Treatment, 0)
	for prescriptionId, prescriptionStatus := range s.PrescriptionIdToPrescriptionStatus {
		medication := &common.Treatment{}
		medication.ErxMedicationId = common.NewObjectId(prescriptionId)
		medication.PrescriptionStatus = prescriptionStatus
		medications = append(medications, medication)
	}
	return medications, nil
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

func (s *StubErxService) GetRefillRequestQueueForClinic() error {
	return nil
}
