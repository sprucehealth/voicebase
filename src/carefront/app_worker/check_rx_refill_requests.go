package app_worker

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/golog"
	"carefront/libs/pharmacy"
	"strconv"
	"time"

	"github.com/samuel/go-metrics/metrics"
)

const (
	waitTimeInMinsForRefillRxChecker = 30 * time.Second
)

func StartWorkerToCheckForRefillRequests(DataApi api.DataAPI, ERxApi erx.ERxAPI, statsRegistry metrics.Registry, environment string) {

	statFailure := metrics.NewCounter()
	statCycles := metrics.NewCounter()

	statsRegistry.Add("cycles/total", statCycles)
	statsRegistry.Add("cycles/failed", statFailure)

	go func() {
		for {

			time.Sleep(waitTimeInMinsForRefillRxChecker)
			PerformRefillRecquestCheckCycle(DataApi, ERxApi, statFailure, statCycles, environment)
		}
	}()
}

func PerformRefillRecquestCheckCycle(DataApi api.DataAPI, ERxApi erx.ERxAPI, statFailure, statCycles metrics.Counter, environment string) {
	// get pending refill request statuses for the clinic that we already have in our database
	refillRequestStatuses, err := DataApi.GetPendingRefillRequestStatusEventsForClinic()
	if err != nil {
		golog.Errorf("Unable to get pending refill request statuses from DB: %+v", refillRequestStatuses)
		statFailure.Inc(1)
		return
	}
	golog.Debugf("Sucessfully made db call to get pending statuses for any existing refill requests. Number of refill requests returned: %d", len(refillRequestStatuses))

	// get refill request queue for clinic
	refillRequestQueue, err := ERxApi.GetRefillRequestQueueForClinic()
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
			if refillRequestStatus.RxRequestQueueItemId == refillRequestItem.RxRequestQueueItemId {
				refillRequestFoundInDB = true
				break
			}
		}

		// noting to do if the refill request already exists
		// in the queue
		if refillRequestFoundInDB {
			continue
		}

		golog.Debugf("Refill request with id %d not found in db, so have to add one", refillRequestItem.RxRequestQueueItemId)

		// identify doctor the refill request belongs to based on clinician id
		doctor, err := DataApi.GetDoctorFromDoseSpotClinicianId(refillRequestItem.ClinicianId)
		if err != nil {
			golog.Errorf("Unable to lookup doctor based on the clinician id: %+v", err)
			statFailure.Inc(1)
			continue
		}

		if doctor == nil {
			golog.Errorf("No doctor exists with this clinician id %d. Need to figure out how best to resolve this error.", refillRequestItem.ClinicianId)
			statFailure.Inc(1)
			continue
		}

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

		refillRequestItem.Doctor = doctor
		golog.Debugf("Doctor identified as %s %s", doctor.FirstName, doctor.LastName)

		// lookup pharmacy associated with prescriptions (dispensed and requested) and link to it
		if err := linkPharmacyToPrescription(DataApi, ERxApi, refillRequestItem.DispensedPrescription); err != nil {
			statFailure.Inc(1)
			continue
		}

		if err := linkPharmacyToPrescription(DataApi, ERxApi, refillRequestItem.RequestedPrescription); err != nil {
			statFailure.Inc(1)
			continue
		}

		// Identify the patient which this refill request is for.
		if refillRequestItem.ErxPatientId == 0 {
			golog.Errorf("Patient to which to map this refill request to not specified. This is an undetermined state.")
			statFailure.Inc(1)
			continue
		}

		patientInDb, err := DataApi.GetPatientFromErxPatientId(refillRequestItem.ErxPatientId)
		if err != nil {
			golog.Errorf("Unable to get patient from db based on erx patient id: %+v", err)
			statFailure.Inc(1)
			continue
		}

		if patientInDb == nil && !refillRequestItem.PatientAddedForRequest && environment == "prod" {
			golog.Errorf("Patient expected to exist in our db but it does not. This is an undetermined state.")
			statFailure.Inc(1)
			continue
		}

		// if patient not yet identified, this is considered an unmatched patient and should be stored in our database so that
		// we can link to this patient information when presenting the refill request to the doctor
		if patientInDb == nil {
			golog.Debugf("Patient does not exist in our system. going to create unlinked patient")

			// get the patient information from dosespot
			patientDetailsFromDoseSpot, err := ERxApi.GetPatientDetails(refillRequestItem.ErxPatientId)
			if err != nil {
				golog.Errorf("Unable to get patient details from dosespot: %+v", err)
				statFailure.Inc(1)
				continue
			}

			err = DataApi.CreateUnlinkedPatientFromRefillRequest(patientDetailsFromDoseSpot)
			if err != nil {
				golog.Errorf("Unable to create unlinked patient in our database: %+v", err)
				statFailure.Inc(1)
				continue
			}
			patientInDb = patientDetailsFromDoseSpot
		} else {
			// match the requested treatment to the original treatment if it exists within our database
			if err := attemptToMatchRequestedPrescriptionToOriginalRx(DataApi, refillRequestItem.RequestedPrescription); err != nil {
				golog.Errorf("Unable to attempt to link requested prescription to originating prescription: %+v", err)
				statFailure.Inc(1)
				continue
			}
		}
		refillRequestItem.Patient = patientInDb

		// Insert refill request into the db. Insert the medication dispensed into its own table in the db, and the
		// requested prescription into its own table as well
		err = DataApi.CreateRefillRequest(refillRequestItem)
		if err != nil {
			golog.Errorf("Unable to store refill request in our database: %+v", err)
			statFailure.Inc(1)
			continue
		}

		// insert queued status into db
		err = DataApi.AddRefillRequestStatusEvent(refillRequestItem.Id, api.RX_REFILL_STATUS_REQUESTED, refillRequestItem.RequestDateStamp)
		if err != nil {
			golog.Errorf("Unable to add refill request event to our database: %+v", err)
			statFailure.Inc(1)
			continue
		}

		// insert refill item into doctor queue as a refill request
		err = DataApi.InsertNewRefillRequestIntoDoctorQueue(refillRequestItem.Id, doctor.DoctorId.Int64())
		if err != nil {
			golog.Errorf("Unable to insert new item into doctor queue that represents the refill request: %+v", err)
			statFailure.Inc(1)
			continue
		}

		golog.Debugf("********************")
	}

	statCycles.Inc(1)
}

func attemptToMatchRequestedPrescriptionToOriginalRx(DataApi api.DataAPI, requestedPrescription *common.Treatment) error {
	return nil
}

func linkPharmacyToPrescription(DataApi api.DataAPI, ERxApi erx.ERxAPI, prescription *common.Treatment) error {
	// lookup pharmacy associated with prescription and link to it
	pharmacyDetails, err := DataApi.GetPharmacyBasedOnReferenceIdAndSource(strconv.FormatInt(prescription.ErxPharmacyId, 10), pharmacy.PHARMACY_SOURCE_SURESCRIPTS)
	if err != nil {
		golog.Errorf("Unable to make a succesful query to lookup pharmacy returned for refill request from our db: %+v", err)
		return err
	}

	if pharmacyDetails == nil {
		golog.Infof("Pharmacy not found in our database. Searched with id %d Getting from surescripts...", prescription.ErxPharmacyId)
		pharmacyDetails, err = ERxApi.GetPharmacyDetails(prescription.ErxPharmacyId)
		if err != nil {
			golog.Errorf("Unable to get pharmacy from surescripts, which means unable to store pharmacy linked to prescription: %+v", err)
			return err
		}
		err = DataApi.AddPharmacy(pharmacyDetails)
		if err != nil {
			golog.Errorf("Unable to store pharmacy in our database: %+v", err)
			return err
		}
	}
	prescription.PharmacyLocalId = common.NewObjectId(pharmacyDetails.LocalId)
	return nil
}
