package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
)

type medicationDispenseUnitsHandler struct {
	dataAPI api.DataAPI
}

func NewMedicationDispenseUnitsHandler(dataAPI api.DataAPI) http.Handler {
	return apiservice.SupportedMethods(&medicationDispenseUnitsHandler{
		dataAPI: dataAPI,
	}, []string{apiservice.HTTP_GET})
}

type MedicationDispenseUnitsResponse struct {
	DispenseUnits []*MedicationDispenseUnitItem `json:"dispense_units"`
}

type MedicationDispenseUnitItem struct {
	Id   int64  `json:"id,string"`
	Text string `json:"text"`
}

func (m *medicationDispenseUnitsHandler) IsAuthorized(r *http.Request) (bool, error) {
	if apiservice.GetContext(r).Role != api.DOCTOR_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (m *medicationDispenseUnitsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dispenseUnitIds, dispenseUnits, err := m.dataAPI.GetMedicationDispenseUnits(api.EN_LANGUAGE_ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
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

	apiservice.WriteJSON(w, medicationDispenseUnitResponse)

}
