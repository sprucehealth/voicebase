package app_worker

import (
	"fmt"
	"time"

	"github.com/samuel/go-metrics/metrics"
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
	waitDurationForRefillRXWorker = 30 * time.Second
)

type RefillRequestWorker struct {
	dataAPI     api.DataAPI
	erxAPI      erx.ERxAPI
	lockAPI     api.LockAPI
	dispatcher  *dispatch.Dispatcher
	statFailure *metrics.Counter
	statCycles  *metrics.Counter
	stopChan    chan bool
}

func NewRefillRequestWorker(
	dataAPI api.DataAPI,
	eRxAPI erx.ERxAPI,
	lockAPI api.LockAPI,
	dispatcher *dispatch.Dispatcher,
	statsRegistry metrics.Registry) *RefillRequestWorker {
	statFailure := metrics.NewCounter()
	statCycles := metrics.NewCounter()

	statsRegistry.Add("cycles/total", statCycles)
	statsRegistry.Add("cycles/failed", statFailure)

	return &RefillRequestWorker{
		dataAPI:     dataAPI,
		erxAPI:      eRxAPI,
		lockAPI:     lockAPI,
		dispatcher:  dispatcher,
		statFailure: statFailure,
		statCycles:  statCycles,
		stopChan:    make(chan bool),
	}

}

func (w *RefillRequestWorker) Start() {
	go func() {
		defer w.lockAPI.Release()
		for {
			if !w.lockAPI.Wait() {
				return
			}

			select {
			case <-w.stopChan:
				return
			default:
			}

			w.Do()

			select {
			case <-w.stopChan:
				return
			case <-time.After(waitDurationForRefillRXWorker):
			}
		}
	}()
}

func (w *RefillRequestWorker) Stop() {
	close(w.stopChan)
}

func (w *RefillRequestWorker) Do() error {
	// Unfortunately, we have to get the clincianId of a doctor to make the call to get refill
	// requests at the clinic level beacuse this call does not work with the proxy clincian Id
	doctor, err := w.dataAPI.GetFirstDoctorWithAClinicianID()
	if err != nil {
		golog.Errorf("Unable to get doctor with clinician id set: %s", err)
		w.statFailure.Inc(1)
		return err
	}

	// get refill request queue for clinic
	refillRequestQueue, err := w.erxAPI.GetRefillRequestQueueForClinic(doctor.DoseSpotClinicianID)
	if err != nil {
		golog.Errorf("Unable to get refill request queue for clinic: %+v", err)
		w.statFailure.Inc(1)
		return err
	}

	// create a map of the queue item id to the refill request
	incomingRefillRequests := make(map[int64]*common.RefillRequestItem)
	// populate a list of the incoming queue item ids
	incomingQueueItemIDs := make([]int64, len(refillRequestQueue))

	for i, refillRequestItem := range refillRequestQueue {
		incomingQueueItemIDs[i] = refillRequestItem.RxRequestQueueItemID
		incomingRefillRequests[refillRequestItem.RxRequestQueueItemID] = refillRequestItem
	}

	// determine a list of non existent incoming refill requests by their queue item id
	nonExistingQueueItemIDs, err := w.dataAPI.FilterOutRefillRequestsThatExist(incomingQueueItemIDs)
	if err != nil {
		golog.Errorf("Unable to filter out existing refill requests: %s", err)
		w.statFailure.Inc(1)
		return err
	}

	// add the new refill requests to the database
	for _, queueItemID := range nonExistingQueueItemIDs {

		refillRequestItem := incomingRefillRequests[queueItemID]

		// Identify the original prescription the refill request links to.
		if refillRequestItem.RequestedPrescription == nil {
			golog.Errorf("Requested prescription does not exist, so no way to approve or deny a refill request that does not exist in complete form")
			w.statFailure.Inc(1)
			continue
		}

		if refillRequestItem.DispensedPrescription == nil {
			golog.Errorf("Dispensed prescription does not exist. Currently assuming this to be an undesired situation, but may not be...")
			w.statFailure.Inc(1)
			continue
		}

		doctor, err := w.dataAPI.GetDoctorFromDoseSpotClinicianID(refillRequestItem.ClinicianID)

		if err != nil {
			golog.Errorf("Unable to get doctor for refill request: %+v", err)
			w.statFailure.Inc(1)
			continue
		}

		if doctor == nil {
			golog.Errorf("No doctor exists with clinician id %d in our system", refillRequestItem.ClinicianID)
			w.statFailure.Inc(1)
			continue
		}
		refillRequestItem.Doctor = doctor

		if err := linkDoctorToPrescription(w.dataAPI, refillRequestItem.RequestedPrescription); err != nil {
			w.statFailure.Inc(1)
			continue
		}

		if err := linkDoctorToPrescription(w.dataAPI, refillRequestItem.DispensedPrescription); err != nil {
			w.statFailure.Inc(1)
			continue
		}

		if refillRequestItem.Doctor.ID.Int64() != refillRequestItem.RequestedPrescription.Doctor.ID.Int64() {
			golog.Errorf("Expected the doctor for the refill request (id = %d) to be the same as the doctor for the requested prescription in the refill request (id = %d), but this is not the case. (refill request queue item id = %d)",
				refillRequestItem.Doctor.ID.Int64(), refillRequestItem.RequestedPrescription.Doctor.ID.Int64(), refillRequestItem.RxRequestQueueItemID)
			w.statFailure.Inc(1)
			continue
		}

		// lookup pharmacy associated with prescriptions (dispensed and requested) and link to it
		if err := linkPharmacyToPrescription(w.dataAPI, w.erxAPI, refillRequestItem.DispensedPrescription); err != nil {
			w.statFailure.Inc(1)
			continue
		}

		if err := linkPharmacyToPrescription(w.dataAPI, w.erxAPI, refillRequestItem.RequestedPrescription); err != nil {
			w.statFailure.Inc(1)
			continue
		}

		// Identify the patient which this refill request is for.
		if refillRequestItem.ErxPatientID == 0 {
			golog.Errorf("Patient to which to map this refill request to not specified. This is an undetermined state.")
			w.statFailure.Inc(1)
			continue
		}

		patientInDB, err := w.dataAPI.GetPatientFromErxPatientID(refillRequestItem.ErxPatientID)
		if err != nil {
			golog.Errorf("Unable to get patient from db based on erx patient id: %+v", err)
			w.statFailure.Inc(1)
			continue
		}

		if patientInDB == nil && !refillRequestItem.PatientAddedForRequest && environment.IsProd() {
			golog.Errorf("Patient expected to exist in our db but it does not. This is an undetermined state.")
			w.statFailure.Inc(1)
			continue
		}

		// if patient not yet identified, this is considered an unmatched patient and should be stored in our database so that
		// we can link to this patient information when presenting the refill request to the doctor
		if patientInDB == nil {
			golog.Debugf("Patient does not exist in our system. going to create unlinked patient")

			// get the patient information from dosespot
			patientDetailsFromDoseSpot, err := w.erxAPI.GetPatientDetails(refillRequestItem.ErxPatientID)
			if err != nil {
				golog.Errorf("Unable to get patient details from dosespot: %+v", err)
				w.statFailure.Inc(1)
				continue
			}

			// TODO: Currently assuming acne for incoming refill requests for an unknown patient. This is
			// most certainly wrong, but for now it's required to link a pathway to the case. Options are
			// making the pathway optional or creating a 'unknown' pathway as a placeholder.
			err = w.dataAPI.CreateUnlinkedPatientFromRefillRequest(patientDetailsFromDoseSpot, doctor, api.AcnePathwayTag)
			if err != nil {
				golog.Errorf("Unable to create unlinked patient in our database: %+v", err)
				w.statFailure.Inc(1)
				continue
			}

			patientInDB = patientDetailsFromDoseSpot
		} else {
			// match the requested treatment to the original treatment if it exists within our database
			if err := w.dataAPI.LinkRequestedPrescriptionToOriginalTreatment(refillRequestItem.RequestedPrescription, patientInDB); err != nil {
				golog.Errorf("Failed attempt at trying to link requested prescription to originating prescription: %+v", err)
				w.statFailure.Inc(1)
				continue
			}
		}
		refillRequestItem.Patient = patientInDB

		// Insert refill request into the db. Insert the medication dispensed into its own table in the db, and the
		// requested prescription into its own table as well
		err = w.dataAPI.CreateRefillRequest(refillRequestItem)
		if err != nil {
			golog.Errorf("Unable to store refill request in our database: %+v", err)
			w.statFailure.Inc(1)
			continue
		}

		// insert queued status into db
		err = w.dataAPI.AddRefillRequestStatusEvent(common.StatusEvent{
			ItemID:            refillRequestItem.ID,
			Status:            api.RXRefillStatusRequested,
			ReportedTimestamp: refillRequestItem.RequestDateStamp,
		})
		if err != nil {
			golog.Errorf("Unable to add refill request event to our database: %+v", err)
			w.statFailure.Inc(1)
			continue
		}

		w.dispatcher.Publish(&RefillRequestCreatedEvent{
			Patient:         refillRequestItem.Patient,
			DoctorID:        refillRequestItem.RequestedPrescription.Doctor.ID.Int64(),
			RefillRequestID: refillRequestItem.ID,
		})

		golog.Debugf("********************")
	}

	w.statCycles.Inc(1)
	return nil
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
	pharmacyDetails, err := dataAPI.GetPharmacyBasedOnReferenceIDAndSource(prescription.ERx.ErxPharmacyID, pharmacy.PharmacySourceSurescripts)
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
	prescription.ERx.PharmacyLocalID = encoding.DeprecatedNewObjectID(pharmacyDetails.LocalID)
	return nil
}
