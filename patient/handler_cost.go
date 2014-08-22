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

type displayLineItem struct {
	Description string `json:"description"`
	Value       string `json:"value"`
}

type costResponse struct {
	LineItems []*displayLineItem `json:"line_items"`
	Total     *displayLineItem   `json:"total"`
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

	itemCost, err := c.dataAPI.GetActiveItemCost(itemType)
	if err == api.NoRowsError {
		apiservice.WriteResourceNotFoundError("no cost found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	costBreakDown := &common.CostBreakdown{LineItems: itemCost.LineItems}
	costBreakDown.CalculateTotal()

	response := costResponse{
		Total: &displayLineItem{
			Value:       costBreakDown.TotalCost.String(),
			Description: "Total",
		},
	}
	for _, lItem := range itemCost.LineItems {
		response.LineItems = append(response.LineItems, &displayLineItem{
			Description: lItem.Description,
			Value:       lItem.Cost.String(),
		})
	}

	apiservice.WriteJSON(w, response)
}
