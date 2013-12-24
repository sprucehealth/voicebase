package apiservice

import (
	"carefront/api"
	"net/http"
)

type MedicationDispenseUnitsHandler struct {
	DataApi api.DataAPI
}

type MedicationDispenseUnitsResponse struct {
	DispenseUnits []*MedicationDispenseUnitItem `json:"dispense_units"`
}

type MedicationDispenseUnitItem struct {
	Id   int64  `json:"id,string"`
	Text string `json:"text"`
}

func (m *MedicationDispenseUnitsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dispenseUnitIds, dispenseUnits, err := m.DataApi.GetMedicationDispenseUnits(api.EN_LANGUAGE_ID)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to retrieve dispense units from database: "+err.Error())
		return
	}
	medicationDispenseUnitResponse := &MedicationDispenseUnitsResponse{}
	medicationDispenseUnitResponse.DispenseUnits = make([]*MedicationDispenseUnitItem, len(dispenseUnits))
	for i, dispenseUnit := range dispenseUnits {
		dispenseUnitItem := &MedicationDispenseUnitItem{}
		dispenseUnitItem.Id = dispenseUnitIds[i]
		dispenseUnitItem.Text = dispenseUnit
		medicationDispenseUnitResponse.DispenseUnits[i] = dispenseUnitItem
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, medicationDispenseUnitResponse)

}
