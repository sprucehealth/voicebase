package cost

import (
	"net/http"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/sku"
)

type costHandler struct {
	dataAPI         api.DataAPI
	analyticsLogger analytics.Logger
}

type displayLineItem struct {
	Description string `json:"description"`
	Value       string `json:"value"`
}

type costResponse struct {
	LineItems []*displayLineItem `json:"line_items"`
	Total     *displayLineItem   `json:"total"`
}

func NewCostHandler(dataAPI api.DataAPI, analyticsLogger analytics.Logger) http.Handler {
	return &costHandler{
		dataAPI:         dataAPI,
		analyticsLogger: analyticsLogger,
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

	accountID := apiservice.GetContext(r).AccountId

	itemType := r.FormValue("item_type")
	if itemType == "" {
		apiservice.WriteValidationError("item_type required", w, r)
		return
	}

	s, err := sku.GetSKU(itemType)
	if err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	costBreakdown, err := totalCostForItems([]sku.SKU{s}, accountID, false, c.dataAPI, c.analyticsLogger)
	if err == api.NoRowsError {
		apiservice.WriteResourceNotFoundError("cost not found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	response := costResponse{
		Total: &displayLineItem{
			Value:       costBreakdown.TotalCost.String(),
			Description: "Total",
		},
	}

	for _, lItem := range costBreakdown.LineItems {
		response.LineItems = append(response.LineItems, &displayLineItem{
			Description: lItem.Description,
			Value:       lItem.Cost.String(),
		})
	}

	apiservice.WriteJSON(w, response)
}
