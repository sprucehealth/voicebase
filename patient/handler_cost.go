package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type costHandler struct {
	dataAPI api.DataAPI
}

func NewCostHandler(dataAPI api.DataAPI) http.Handler {
	return &costHandler{
		dataAPI: dataAPI,
	}
}

func (c *costHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.PATIENT_ROLE {
		return false, nil
	}

	return true, nil
}

func (c *costHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_GET {
		http.NotFound(w, r)
		return
	}

	itemType := r.FormValue("item_type")
	if itemType == "" {
		apiservice.WriteValidationError("item_type required", w, r)
		return
	}

	lineItems, err := c.dataAPI.GetLineItemsForType(itemType)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	costBreakDown := &common.CostBreakdown{LineItems: lineItems}
	costBreakDown.CalculateTotal()

	apiservice.WriteJSON(w, costBreakDown)
}
