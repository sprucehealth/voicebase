package apiservice

import (
	"carefront/api"
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/pharmacy"
	"net/http"
	"strconv"
)

type DoctorPrescriptionsErrorsHandler struct {
	DataApi api.DataAPI
	ErxApi  erx.ERxAPI
}

type DoctorPrescriptionErrorsResponse struct {
	TransmissionErrors []*transmissionErrorItem `json:"transmission_errors"`
}

type transmissionErrorItem struct {
	Treatment *common.Treatment      `json:"treatment,omitempty"`
	Patient   *common.Patient        `json:"patient,omitempty"`
	Pharmacy  *pharmacy.PharmacyData `json:"pharmacy,omitempty"`
}

func (d *DoctorPrescriptionsErrorsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	doctor, err := d.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	medicationsWithErrors, err := d.ErxApi.GetTransmissionErrorDetails(doctor.DoseSpotClinicianId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get prescription related errors: "+err.Error())
		return
	}

	transmissionErrors := make([]*transmissionErrorItem, 0)
	uniquePatientIdsBookKeeping := make(map[int64]bool)
	uniquePatientIds := make([]int64, 0)
	pharmacyIdToTransmissionErrorMapping := make(map[int64]*transmissionErrorItem)
	for _, medicationWithError := range medicationsWithErrors {
		treatment, err := d.DataApi.GetTreatmentBasedOnPrescriptionId(medicationWithError.PrescriptionId.Int64())
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get treatment based on prescription: "+err.Error())
		}

		// there is no treatment in our system based on the prescription.
		// lets not ignore the error, instead lets just show the data as we have it from dosespot
		// without linking it to patient data. This can happen in the event that the prescription id
		// did not get stored for some reason or if we have multiple doctors using the same account (incorrectly)
		if treatment == nil {
			treatment = medicationWithError
		} else {
			if !uniquePatientIdsBookKeeping[treatment.PatientId.Int64()] {
				uniquePatientIdsBookKeeping[treatment.PatientId.Int64()] = true
				uniquePatientIds = append(uniquePatientIds, treatment.PatientId.Int64())
			}
		}

		treatment.PrescriptionStatus = medicationWithError.PrescriptionStatus
		treatment.TransmissionErrorDate = medicationWithError.TransmissionErrorDate
		treatment.StatusDetails = medicationWithError.StatusDetails
		if treatment.ErxSentDate == nil {
			treatment.ErxSentDate = medicationWithError.ErxSentDate
		}

		transmissionError := &transmissionErrorItem{
			Treatment: treatment,
		}

		// keep track of which pharmacy Id maps to which transmissionError so that we can assign the pharmacy to the transmissionError
		pharmacyIdToTransmissionErrorMapping[medicationWithError.ErxPharmacyId] = transmissionError
		transmissionErrors = append(transmissionErrors, transmissionError)
	}

	patients, err := d.DataApi.GetPatientsForIds(uniquePatientIds)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get patients based on ids: "+err.Error())
		return
	}

	pharmacies, err := d.DataApi.GetPharmacySelectionForPatients(uniquePatientIds)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get pharmacies for patients based on ids: "+err.Error())
		return
	}

	for pharmacyId, transmissionError := range pharmacyIdToTransmissionErrorMapping {
		// check if the pharmacy exists in the pharmacies returned
		for _, pharmacySelection := range pharmacies {
			pharmacyIdInt, _ := strconv.ParseInt(pharmacySelection.Id, 0, 64)
			if pharmacySelection.Source != pharmacy.PHARMACY_SOURCE_SURESCRIPTS && pharmacyIdInt == pharmacyId {
				transmissionError.Pharmacy = pharmacySelection
			} else {
				// TODO lookup pharmacy from surescripts based on id and assign it here
			}
		}
	}

	for _, pharmacySelection := range pharmacies {
		for _, patient := range patients {
			if patient.PatientId.Int64() == pharmacySelection.PatientId {
				patient.Pharmacy = pharmacySelection
			}
		}
	}

	for _, transmissionError := range transmissionErrors {
		for _, patient := range patients {
			if patient.PatientId == transmissionError.Treatment.PatientId {
				transmissionError.Patient = patient
			}
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPrescriptionErrorsResponse{TransmissionErrors: transmissionErrors})
}
