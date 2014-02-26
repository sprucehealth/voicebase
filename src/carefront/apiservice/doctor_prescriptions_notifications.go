package apiservice

import (
	"carefront/api"
	"carefront/libs/erx"
	"net/http"
)

type DoctorPrescriptionsNotificationsHandler struct {
	DataApi api.DataAPI
	ErxApi  erx.ERxAPI
}

type DoctorPrescriptionsNotificationResponse struct {
	RefillRequestsCount    int64 `json:"refill_requests,string,omitempty"`
	TransactionErrorsCount int64 `json:"errors,string,omitempty"`
}

func (d *DoctorPrescriptionsNotificationsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	doctor, err := d.DataApi.GetDoctorFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	refillRequests, transactionErrors, err := d.ErxApi.GetTransmissionErrorRefillRequestsCount(doctor.DoseSpotClinicianId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get notifications count: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPrescriptionsNotificationResponse{
		RefillRequestsCount:    refillRequests,
		TransactionErrorsCount: transactionErrors,
	})
}
