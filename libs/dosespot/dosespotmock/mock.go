package dosespotmock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/dosespot"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ dosespot.API = &Client{}

type Client struct {
	*mock.Expector
}

func New(t *testing.T) *Client {
	return &Client{
		&mock.Expector{T: t},
	}
}

func (c *Client) ApproveRefillRequest(clinicianID, erxRefillRequestQueueItemID, approvedRefillAmount int64, comments string) (int64, error) {
	rets := c.Record(clinicianID, erxRefillRequestQueueItemID, approvedRefillAmount, comments)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (c *Client) DenyRefillRequest(clinicianID, erxRefillRequestQueueItemID int64, denialReason, comments string) (int64, error) {
	rets := c.Record(clinicianID, erxRefillRequestQueueItemID, denialReason, comments)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (c *Client) GetDrugNamesForDoctor(clinicianID int64, prefix string) ([]string, error) {
	rets := c.Record(clinicianID, prefix)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]string), mock.SafeError(rets[1])
}

func (c *Client) GetDrugNamesForPatient(prefix string) ([]string, error) {
	rets := c.Record(prefix)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]string), mock.SafeError(rets[1])
}

func (c *Client) GetPatientDetails(erxPatientID int64) (*dosespot.PatientUpdate, error) {
	rets := c.Record(erxPatientID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dosespot.PatientUpdate), mock.SafeError(rets[1])
}

func (c *Client) GetPharmacyDetails(pharmacyID int64) (*dosespot.Pharmacy, error) {
	rets := c.Record(pharmacyID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dosespot.Pharmacy), mock.SafeError(rets[1])
}

func (c *Client) GetPrescriptionStatus(clinicianID, prescriptionID int64) ([]*dosespot.PrescriptionLogInfo, error) {
	rets := c.Record(clinicianID, prescriptionID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dosespot.PrescriptionLogInfo), mock.SafeError(rets[1])
}

func (c *Client) GetRefillRequestQueueForClinic(clinicianID int64) ([]*dosespot.RefillRequestQueueItem, error) {
	rets := c.Record(clinicianID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dosespot.RefillRequestQueueItem), mock.SafeError(rets[1])
}

func (c *Client) GetTransmissionErrorDetails(clinicianID int64) ([]*dosespot.TransmissionErrorDetails, error) {
	rets := c.Record(clinicianID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dosespot.TransmissionErrorDetails), mock.SafeError(rets[1])
}

func (c *Client) GetTransmissionErrorRefillRequestsCount(clinicianID int64) (refillRequests int64, transactionErrors int64, err error) {
	rets := c.Record(clinicianID)
	if len(rets) == 0 {
		return 0, 0, nil
	}
	return rets[0].(int64), rets[1].(int64), mock.SafeError(rets[2])
}

func (c *Client) IgnoreAlert(clinicianID int64, prescriptionID int64) error {
	rets := c.Record(clinicianID, prescriptionID)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (c *Client) SearchForAllergyRelatedMedications(searchTerm string) ([]string, error) {
	rets := c.Record(searchTerm)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]string), mock.SafeError(rets[1])
}

func (c *Client) SearchForMedicationStrength(clinicianID int64, medicationName string) ([]string, error) {
	rets := c.Record(clinicianID, medicationName)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]string), mock.SafeError(rets[1])
}

func (c *Client) SearchForPharmacies(clinicianID int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*dosespot.Pharmacy, error) {
	rets := c.Record(clinicianID, city, state, zipcode, name, pharmacyTypes)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dosespot.Pharmacy), mock.SafeError(rets[1])
}

func (c *Client) SelectMedication(clinicianID int64, medicationName, medicationStrength string) (*dosespot.MedicationSelectResponse, error) {
	rets := c.Record(clinicianID, medicationName, medicationStrength)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dosespot.MedicationSelectResponse), mock.SafeError(rets[1])
}

func (c *Client) SendMultiplePrescriptions(clinicianID, eRxPatientID int64, prescriptionIDs []int64) ([]*dosespot.SendPrescriptionResult, error) {
	rets := c.Record(clinicianID, eRxPatientID, prescriptionIDs)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dosespot.SendPrescriptionResult), mock.SafeError(rets[1])
}

func (c *Client) StartPrescribingPatient(clinicianID int64, patient *dosespot.Patient, prescriptions []*dosespot.Prescription, pharmacySourceID int64) ([]*dosespot.PatientUpdate, error) {
	rets := c.Record(clinicianID, patient, prescriptions, pharmacySourceID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dosespot.PatientUpdate), mock.SafeError(rets[1])
}

func (c *Client) UpdatePatientInformation(clinicianID int64, patient *dosespot.Patient, pharmacyID int64) ([]*dosespot.PatientUpdate, error) {
	rets := c.Record(clinicianID, patient, pharmacyID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].([]*dosespot.PatientUpdate), mock.SafeError(rets[1])
}
