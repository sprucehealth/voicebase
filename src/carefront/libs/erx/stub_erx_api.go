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

func (s *StubErxService) GetDrugNamesForDoctor(prefix string) ([]string, error) {
	return nil, nil
}

func (s *StubErxService) GetDrugNamesForPatient(prefix string) ([]string, error) {
	return nil, nil
}

func (s *StubErxService) SearchForMedicationStrength(medicationName string) ([]string, error) {
	return nil, nil
}

func (s *StubErxService) SelectMedication(medicationName, medicationStrength string) (medication *Medication, err error) {
	return nil, nil
}

func (s *StubErxService) StartPrescribingPatient(Patient *common.Patient, Treatments []*common.Treatment) error {
	fmt.Println("Starting to prescribe patient")
	// walk through the treatments and assign them each a prescription id
	// assumption here is that there are as many prescription ids to return as there are treatments
	Patient.ERxPatientId = common.NewObjectId(s.PatientErxId)
	for i, treatment := range Treatments {
		treatment.PrescriptionId = common.NewObjectId(s.PrescriptionIdsToReturn[i])
	}
	return nil
}

func (s *StubErxService) SendMultiplePrescriptions(Patient *common.Patient, Treatments []*common.Treatment) ([]int64, error) {
	// nothing to do here given that the act of sending a prescription successfully does not change the state of the system
	fmt.Println("Sending multiple prescriptions")
	return nil, nil
}

func (s *StubErxService) SearchForPharmacies(city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacySearch.PharmacyData, error) {
	return nil, nil
}

func (s *StubErxService) GetPrescriptionStatus(prescriptionId int64) ([]*PrescriptionLog, error) {
	return nil, nil
}

func (s *StubErxService) GetMedicationList(PatientId int64) ([]*Medication, error) {
	medications := make([]*Medication, 0)
	for prescriptionId, prescriptionStatus := range s.PrescriptionIdToPrescriptionStatus {
		medication := &Medication{}
		medication.ErxMedicationId = prescriptionId
		medication.PrescriptionStatus = prescriptionStatus
		medications = append(medications, medication)
	}
	return medications, nil
}

func (s *StubErxService) GetTransmissionErrorDetails() ([]*Medication, error) {
	return nil, nil
}

func (s *StubErxService) GetTransmissionErrorRefillRequestsCount() (refillRequests int64, transactionErrors int64, err error) {
	return
}

func (s *StubErxService) IgnoreAlert(prescriptionId int64) error {
	return nil
}
