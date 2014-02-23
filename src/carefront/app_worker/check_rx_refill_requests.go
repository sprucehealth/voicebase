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

func StartWorkerToCheckForRefillRequests(DataApi api.DataAPI, ERxApi erx.ERxAPI, statsRegistry metrics.Registry) {

	statFailure := metrics.NewCounter()
	statCycles := metrics.NewCounter()

	statsRegistry.Add("cycles/total", statCycles)
	statsRegistry.Add("cycles/failed", statFailure)

	go func() {
		for {

			time.Sleep(waitTimeInMinsForRefillRxChecker)
			PerformRefillRecquestCheckCycle(DataApi, ERxApi, statFailure, statCycles)
		}
	}()
}

func PerformRefillRecquestCheckCycle(DataApi api.DataAPI, ERxApi erx.ERxAPI, statFailure, statCycles metrics.Counter) {
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
			golog.Errorf("Requested prescription does not exist, so no way to link this to the original prescription")
			statFailure.Inc(1)
			continue
		}

		if refillRequestItem.DispensedPrescription == nil {
			golog.Errorf("Dispensed prescription does not exist. Currently assuming this to be an undesired situation, but may not be...")
			statFailure.Inc(1)
			continue
		}

		if refillRequestItem.RequestedPrescription.ErxPharmacyId != refillRequestItem.DispensedPrescription.ErxPharmacyId {
			golog.Errorf("The pharmacy information betwee the requested and dispensed prescriptions are different, when this should not be the case.")
			statFailure.Inc(1)
			continue
		}

		refillRequestItem.Doctor = doctor
		golog.Debugf("Doctor identified as %s %s", doctor.FirstName, doctor.LastName)

		// lookup pharmacy associated with prescription and link to it
		pharmacyDetails, err := DataApi.GetPharmacyBasedOnReferenceIdAndSource(strconv.FormatInt(refillRequestItem.RequestedPrescription.ErxPharmacyId, 10), pharmacy.PHARMACY_SOURCE_SURESCRIPTS)
		if err != nil {
			golog.Errorf("Unable to make a succesful query to lookup pharmacy returned for refill request from our db: %+v", err)
			statFailure.Inc(1)
			continue
		}

		if pharmacyDetails == nil {
			golog.Infof("Pharmacy that the original prescription links to is not found in our database. Searched with id %d Getting from surescripts...", refillRequestItem.RequestedPrescription.ErxPharmacyId)
			pharmacyDetails, err = ERxApi.GetPharmacyDetails(refillRequestItem.RequestedPrescription.ErxPharmacyId)
			if err != nil {
				golog.Errorf("Unable to get pharmacy from surescripts, which means unable to store pharmacy and link to original prescription: %+v", err)
				statFailure.Inc(1)
				continue
			}
			err = DataApi.AddPharmacy(pharmacyDetails)
			if err != nil {
				golog.Errorf("Unable to store pharmacy in our database: %+v", err)
				statFailure.Inc(1)
				continue
			}
		}
		refillRequestItem.DispensedPrescription.PharmacyLocalId = common.NewObjectId(pharmacyDetails.LocalId)
		refillRequestItem.RequestedPrescription.PharmacyLocalId = common.NewObjectId(pharmacyDetails.LocalId)
		golog.Debugf("Pharmacy identified in our db as %d", pharmacyDetails.LocalId)

		originalTreatment, err := DataApi.GetTreatmentBasedOnPrescriptionId(refillRequestItem.RequestedPrescription.PrescriptionId.Int64())
		if err != nil {
			golog.Errorf("Unable to lookup original prescription %+v", err)
			statFailure.Inc(1)
			continue
		}

		if originalTreatment == nil {
			golog.Debugf(`Original treatment with prescription id %d does not exist in our database. Going to create an unlinked treatment in our db`, refillRequestItem.RequestedPrescription.PrescriptionId.Int64())

			// if the treatment does not exist in our system, lets go ahead and create an unlinked treatment
			refillRequestItem.UnlinkedRequestedPrescription = refillRequestItem.RequestedPrescription
			err = DataApi.AddUnlinkedTreatmentFromPharmacy(refillRequestItem.RequestedPrescription)
			if err != nil {
				golog.Errorf("Original prescription does not exist in our system, and we were unable to create it as an unlinked treatment in our system: %+v", err)
				statFailure.Inc(1)
				continue
			}
			originalTreatment = refillRequestItem.RequestedPrescription
			refillRequestItem.UnlinkedRequestedPrescription = refillRequestItem.RequestedPrescription
			refillRequestItem.RequestedPrescription = nil
		} else {
			// assigning the treatment plan id to the requested prescription with the assumption that
			if !originalTreatment.Equals(refillRequestItem.RequestedPrescription) {
				golog.Errorf(`Original treatment returned from database does not match requested prescription from dosespot. 
							This is an inconsistent state and should not happen.`)
				statFailure.Inc(1)
				continue
			}
			refillRequestItem.RequestedPrescription.Id = originalTreatment.Id
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

		if patientInDb == nil && !refillRequestItem.PatientAddedForRequest {
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
		}
		refillRequestItem.Patient = patientInDb

		// Insert refill request into the db. Insert the medication dispensed into its own table in the db, against the original
		// treatment (which in turn link to the patient visit and the treatment plan)
		err = DataApi.CreateRefillRequest(refillRequestItem)
		if err != nil {
			golog.Errorf("Unable to store refill request in our database: %+v", err)
			statFailure.Inc(1)
			continue
		}

		// insert queued status into db
		err = DataApi.AddRefillRequestStatusEvent(refillRequestItem.Id, api.RX_REFILL_STATUS_QUEUED, refillRequestItem.RequestDateStamp)
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
