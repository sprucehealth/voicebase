package app_worker

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/pharmacy"
)

const (
	waitTimeInMinsForRefillRxChecker = 30 * time.Second
)

func StartWorkerToCheckForRefillRequests(dataAPI api.DataAPI, eRxAPI erx.ERxAPI, dispatcher *dispatch.Dispatcher, statsRegistry metrics.Registry) {
	statFailure := metrics.NewCounter()
	statCycles := metrics.NewCounter()

	statsRegistry.Add("cycles/total", statCycles)
	statsRegistry.Add("cycles/failed", statFailure)

	go func() {
		for {

			time.Sleep(waitTimeInMinsForRefillRxChecker)
			PerformRefillRecquestCheckCycle(dataAPI, eRxAPI, dispatcher, statFailure, statCycles)
		}
	}()
}

func PerformRefillRecquestCheckCycle(dataAPI api.DataAPI, eRxAPI erx.ERxAPI, dispatcher *dispatch.Dispatcher, statFailure, statCycles *metrics.Counter) {
	// get pending refill request statuses for the clinic that we already have in our database
	refillRequestStatuses, err := dataAPI.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		golog.Errorf("Unable to get pending refill request statuses from DB: %+v", err)
		statFailure.Inc(1)
		return
	}
	golog.Debugf("Sucessfully made db call to get pending statuses for any existing refill requests. Number of refill requests returned: %d", len(refillRequestStatuses))

	// Unfortunately, we have to get the clincianId of a doctor to make the call to get refill
	// requests at the clinic level beacuse this call does not work with the proxy clincian Id
	doctor, err := dataAPI.GetFirstDoctorWithAClinicianID()
	if err != nil {
		golog.Errorf("Unable to get doctor with clinician id set: %s", err)
		statFailure.Inc(1)
		return
	}

	// get refill request queue for clinic
	refillRequestQueue, err := eRxAPI.GetRefillRequestQueueForClinic(doctor.DoseSpotClinicianID)
	if err != nil {
		golog.Errorf("Unable to get refill request queue for clinic: %+v", err)
		statFailure.Inc(1)
		return
	}
	golog.Debugf("Sucessfully made call to get refill requests. Number of refill requests returned: %d", len(refillRequestQueue))

	// determine any new refill requests
	for _, refillRequestItem := range refillRequestQueue {

		refillRequestFoundInDB := false
		for _, refillRequestStatus := range refillRequestStatuses {
			refillRequest, err := dataAPI.GetRefillRequestFromID(refillRequestStatus.ItemID)
			if err != nil {
				golog.Errorf("Unable to get refill request based on id: %+v", err)
				statFailure.Inc(1)
				return
			}

			if refillRequest.RxRequestQueueItemID == refillRequestItem.RxRequestQueueItemID {
				refillRequestFoundInDB = true
				break
			}
		}

		// noting to do if the refill request already exists
		// in the queue
		if refillRequestFoundInDB {
			continue
		}

		golog.Debugf("Refill request with id %d not found in db, so have to add one", refillRequestItem.RxRequestQueueItemID)

		// Identify the original prescription the refill request links to.
		if refillRequestItem.RequestedPrescription == nil {
			golog.Errorf("Requested prescription does not exist, so no way to approve or deny a refill request that does not exist in complete form")
			statFailure.Inc(1)
			continue
		}

		if refillRequestItem.DispensedPrescription == nil {
			golog.Errorf("Dispensed prescription does not exist. Currently assuming this to be an undesired situation, but may not be...")
			statFailure.Inc(1)
			continue
		}

		doctor, err := dataAPI.GetDoctorFromDoseSpotClinicianID(refillRequestItem.ClinicianID)

		if err != nil {
			golog.Errorf("Unable to get doctor for refill request: %+v", err)
			statFailure.Inc(1)
			continue
		}

		if doctor == nil {
			golog.Errorf("No doctor exists with clinician id %d in our system", refillRequestItem.ClinicianID)
			statFailure.Inc(1)
			continue
		}
		refillRequestItem.Doctor = doctor

		if err := linkDoctorToPrescription(dataAPI, refillRequestItem.RequestedPrescription); err != nil {
			statFailure.Inc(1)
			continue
		}

		if err := linkDoctorToPrescription(dataAPI, refillRequestItem.DispensedPrescription); err != nil {
			statFailure.Inc(1)
			continue
		}

		if refillRequestItem.Doctor.DoctorID.Int64() != refillRequestItem.RequestedPrescription.Doctor.DoctorID.Int64() {
			golog.Errorf("Expected the doctor for the refill request (id = %d) to be the same as the doctor for the requested prescription in the refill request (id = %d), but this is not the case. (refill request queue item id = %d)", refillRequestItem.Doctor.DoctorID.Int64(),
				refillRequestItem.RequestedPrescription.Doctor.DoctorID.Int64(), refillRequestItem.RxRequestQueueItemID)
			statFailure.Inc(1)
			continue
		}

		// lookup pharmacy associated with prescriptions (dispensed and requested) and link to it
		if err := linkPharmacyToPrescription(dataAPI, eRxAPI, refillRequestItem.DispensedPrescription); err != nil {
			statFailure.Inc(1)
			continue
		}

		if err := linkPharmacyToPrescription(dataAPI, eRxAPI, refillRequestItem.RequestedPrescription); err != nil {
			statFailure.Inc(1)
			continue
		}

		// Identify the patient which this refill request is for.
		if refillRequestItem.ErxPatientID == 0 {
			golog.Errorf("Patient to which to map this refill request to not specified. This is an undetermined state.")
			statFailure.Inc(1)
			continue
		}

		patientInDb, err := dataAPI.GetPatientFromErxPatientID(refillRequestItem.ErxPatientID)
		if err != nil {
			golog.Errorf("Unable to get patient from db based on erx patient id: %+v", err)
			statFailure.Inc(1)
			continue
		}

		if patientInDb == nil && !refillRequestItem.PatientAddedForRequest && environment.IsProd() {
			golog.Errorf("Patient expected to exist in our db but it does not. This is an undetermined state.")
			statFailure.Inc(1)
			continue
		}

		// if patient not yet identified, this is considered an unmatched patient and should be stored in our database so that
		// we can link to this patient information when presenting the refill request to the doctor
		if patientInDb == nil {
			golog.Debugf("Patient does not exist in our system. going to create unlinked patient")

			// get the patient information from dosespot
			patientDetailsFromDoseSpot, err := eRxAPI.GetPatientDetails(refillRequestItem.ErxPatientID)
			if err != nil {
				golog.Errorf("Unable to get patient details from dosespot: %+v", err)
				statFailure.Inc(1)
				continue
			}

			err = dataAPI.CreateUnlinkedPatientFromRefillRequest(patientDetailsFromDoseSpot, doctor, api.HEALTH_CONDITION_ACNE_ID)
			if err != nil {
				golog.Errorf("Unable to create unlinked patient in our database: %+v", err)
				statFailure.Inc(1)
				continue
			}

			patientInDb = patientDetailsFromDoseSpot
		} else {
			// match the requested treatment to the original treatment if it exists within our database
			if err := dataAPI.LinkRequestedPrescriptionToOriginalTreatment(refillRequestItem.RequestedPrescription, patientInDb); err != nil {
				golog.Errorf("Failed attempt at trying to link requested prescription to originating prescription: %+v", err)
				statFailure.Inc(1)
				continue
			}
		}
		refillRequestItem.Patient = patientInDb

		// Insert refill request into the db. Insert the medication dispensed into its own table in the db, and the
		// requested prescription into its own table as well
		err = dataAPI.CreateRefillRequest(refillRequestItem)
		if err != nil {
			golog.Errorf("Unable to store refill request in our database: %+v", err)
			statFailure.Inc(1)
			continue
		}

		// insert queued status into db
		err = dataAPI.AddRefillRequestStatusEvent(common.StatusEvent{
			ItemID:            refillRequestItem.ID,
			Status:            api.RX_REFILL_STATUS_REQUESTED,
			ReportedTimestamp: refillRequestItem.RequestDateStamp,
		})
		if err != nil {
			golog.Errorf("Unable to add refill request event to our database: %+v", err)
			statFailure.Inc(1)
			continue
		}

		dispatcher.Publish(&RefillRequestCreatedEvent{
			DoctorID:        refillRequestItem.RequestedPrescription.Doctor.DoctorID.Int64(),
			RefillRequestID: refillRequestItem.ID,
		})

		golog.Debugf("********************")
	}

	statCycles.Inc(1)
}

func linkDoctorToPrescription(dataAPI api.DataAPI, prescription *common.Treatment) error {
	// identify doctor the prescription belongs to based on clinician id
	doctor, err := dataAPI.GetDoctorFromDoseSpotClinicianID(prescription.ERx.DoseSpotClinicianID)
	if err != nil {
		golog.Errorf("Unable to lookup doctor based on the clinician id: %+v", err)
		return err
	}

	if doctor == nil {
		golog.Errorf("No doctor exists with this clinician id %d. Need to figure out how best to resolve this error.", prescription.ERx.DoseSpotClinicianID)
		return fmt.Errorf("No doctor exists with clinician id %d in our system", prescription.ERx.DoseSpotClinicianID)
	}

	prescription.Doctor = doctor
	return nil
}

func linkPharmacyToPrescription(dataAPI api.DataAPI, eRxAPI erx.ERxAPI, prescription *common.Treatment) error {
	// lookup pharmacy associated with prescription and link to it
	pharmacyDetails, err := dataAPI.GetPharmacyBasedOnReferenceIdAndSource(prescription.ERx.ErxPharmacyID, pharmacy.PHARMACY_SOURCE_SURESCRIPTS)
	if err != nil {
		golog.Errorf("Unable to make a succesful query to lookup pharmacy returned for refill request from our db: %+v", err)
		return err
	}

	if pharmacyDetails == nil {
		golog.Infof("Pharmacy not found in our database. Searched with id %d Getting from surescripts...", prescription.ERx.ErxPharmacyID)
		pharmacyDetails, err = eRxAPI.GetPharmacyDetails(prescription.ERx.ErxPharmacyID)
		if err != nil {
			golog.Errorf("Unable to get pharmacy from surescripts, which means unable to store pharmacy linked to prescription: %+v", err)
			return err
		}
		err = dataAPI.AddPharmacy(pharmacyDetails)
		if err != nil {
			golog.Errorf("Unable to store pharmacy in our database: %+v", err)
			return err
		}
	}
	prescription.ERx.PharmacyLocalID = encoding.NewObjectID(pharmacyDetails.LocalID)
	return nil
}
