package doctor_treatment_plan

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/pharmacy"
)

const (
	successful_erx_routing_pharmacy_id = 47731
)

func routeRxInTreatmentPlanToPharmacy(treatmentPlanId int64, patient *common.Patient, doctor *common.Doctor,
	routeErx bool, dataAPI api.DataAPI, erxAPI erx.ERxAPI, erxStatusQueue *common.SQSQueue) error {

	// FIX: Remove once we start accepting surescripts pharmacies from patient
	if patient.Pharmacy == nil || patient.Pharmacy.Source != pharmacy.PHARMACY_SOURCE_SURESCRIPTS {
		patient.Pharmacy = &pharmacy.PharmacyData{
			SourceId:     successful_erx_routing_pharmacy_id,
			Source:       pharmacy.PHARMACY_SOURCE_SURESCRIPTS,
			AddressLine1: "123 TEST TEST",
			City:         "San Francisco",
			State:        "CA",
			Postal:       "94115",
		}
	}

	treatments, err := dataAPI.GetTreatmentsBasedOnTreatmentPlanId(treatmentPlanId)
	if err != nil {
		return err
	}

	if !routeErx || erxAPI == nil || len(treatments) == 0 {
		return nil
	}

	// Send the patient information and new medications to be prescribed, to dosespot
	if err := erxAPI.StartPrescribingPatient(doctor.DoseSpotClinicianId, patient, treatments, patient.Pharmacy.SourceId); err != nil {
		return err
	}

	// Save erx patient id to database once we get it back from dosespot
	if err := dataAPI.UpdatePatientWithERxPatientId(patient.PatientId.Int64(), patient.ERxPatientId.Int64()); err != nil {
		return err
	}

	// Save prescription ids for drugs to database
	if err := dataAPI.UpdateTreatmentWithPharmacyAndErxId(treatments, patient.Pharmacy, doctor.DoctorId.Int64()); err != nil {
		return err
	}

	// Now, request the medications to be sent to the patient's preferred pharmacy
	unSuccessfulTreatments, err := erxAPI.SendMultiplePrescriptions(doctor.DoseSpotClinicianId, patient, treatments)
	if err != nil {
		return err
	}

	// gather treatmentIds for treatments that were successfully routed to pharmacy
	successfulTreatments := make([]*common.Treatment, 0, len(treatments))
	for _, treatment := range treatments {
		treatmentFound := false
		for _, unSuccessfulTreatment := range unSuccessfulTreatments {
			if unSuccessfulTreatment.Id.Int64() == treatment.Id.Int64() {
				treatmentFound = true
				break
			}
		}
		if !treatmentFound {
			successfulTreatments = append(successfulTreatments, treatment)
		}
	}

	if err := dataAPI.AddErxStatusEvent(successfulTreatments, common.StatusEvent{Status: api.ERX_STATUS_SENDING}); err != nil {
		return err
	}

	if err := dataAPI.AddErxStatusEvent(unSuccessfulTreatments, common.StatusEvent{Status: api.ERX_STATUS_SEND_ERROR}); err != nil {
		return err
	}

	//  Queue up notification to patient
	if err := apiservice.QueueUpJob(erxStatusQueue, &common.PrescriptionStatusCheckMessage{
		PatientId:      patient.PatientId.Int64(),
		DoctorId:       doctor.DoctorId.Int64(),
		EventCheckType: common.ERxType,
	}); err != nil {
		golog.Errorf("Unable to enqueue job to check status of erx. Not going to error out on this for the user because there is nothing the user can do about this: %+v", err)
	}

	return nil
}
