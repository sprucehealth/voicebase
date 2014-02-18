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
	refillRequests, transactionErrors, err := d.ErxApi.GetTransmissionErrorRefillRequestsCount()
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get notifications count: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorPrescriptionsNotificationResponse{
		RefillRequestsCount:    refillRequests,
		TransactionErrorsCount: transactionErrors,
	})
}
