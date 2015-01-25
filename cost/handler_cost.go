package cost

import (
	"net/http"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/sku"
)

type costHandler struct {
	dataAPI         api.DataAPI
	analyticsLogger analytics.Logger
}

type displayLineItem struct {
	Description string `json:"description"`
	Value       string `json:"value"`
	ChargeValue string `json:"charge_value"`
	Currency    string `json:"currency"`
}

type costResponse struct {
	LineItems []*displayLineItem `json:"line_items"`
	Total     *displayLineItem   `json:"total"`
}

func NewCostHandler(dataAPI api.DataAPI, analyticsLogger analytics.Logger) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(&costHandler{
				dataAPI:         dataAPI,
				analyticsLogger: analyticsLogger,
			}), []string{api.PATIENT_ROLE}), []string{"GET"})
}

func (c *costHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accountID := apiservice.GetContext(r).AccountID

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
	if api.IsErrNotFound(err) {
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
			ChargeValue: costBreakdown.TotalCost.Charge(),
			Currency:    costBreakdown.TotalCost.Currency,
		},
	}

	for _, lItem := range costBreakdown.LineItems {
		response.LineItems = append(response.LineItems, &displayLineItem{
			Description: lItem.Description,
			Value:       lItem.Cost.String(),
			ChargeValue: lItem.Cost.Charge(),
			Currency:    lItem.Cost.Currency,
		})
	}

	apiservice.WriteJSON(w, response)
}
