package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
	"net/http"
	"strconv"
)

type TreatmentsHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type TreatmentsResponse struct {
	TreatmentIds []string `json:"treatment_ids"`
}

type TreatmentsRequestBody struct {
	Treatments []*common.Treatment `json:"treatments"`
}

func NewTreatmentsHandler(dataApi api.DataAPI) *TreatmentsHandler {
	return &TreatmentsHandler{dataApi, 0}
}

func (t *TreatmentsHandler) AccountIdFromAuthToken(accountId int64) {
	t.accountId = accountId
}

func (t *TreatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	treatmentsRequestBody := &TreatmentsRequestBody{}

	err := jsonDecoder.Decode(treatmentsRequestBody)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatment body: "+err.Error())
		return
	}

	if len(treatmentsRequestBody.Treatments) == 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "Nothing to do becuase no treatments were passed to add: "+err.Error())
		return
	}

	// just to be on the safe side, verify each of the treatments that the doctor is trying to add
	for _, treatment := range treatmentsRequestBody.Treatments {
		_, _, _, httpStatusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(treatment.PatientVisitId, t.accountId, t.DataApi)
		if err != nil {
			WriteDeveloperError(w, httpStatusCode, "Unable to validate doctor to add treatment to patient visit: "+err.Error())
			return
		}
	}

	// TODO  validate treatment object

	// Add treatments to patient
	err = t.DataApi.AddTreatmentsForPatientVisit(treatmentsRequestBody.Treatments)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add treatment to patient visit: "+err.Error())
		return
	}

	treatmentIds := make([]string, 0)
	for _, treatment := range treatmentsRequestBody.Treatments {
		treatmentIds = append(treatmentIds, strconv.FormatInt(treatment.Id, 10))
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &TreatmentsResponse{TreatmentIds: treatmentIds})
}
