package cost

import (
	"net/http"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
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

	skuType := r.FormValue("item_type")
	if skuType == "" {
		apiservice.WriteValidationError("item_type required", w, r)
		return
	}

	costBreakdown, err := totalCostForItems([]string{skuType}, accountID, false, c.dataAPI, c.analyticsLogger)
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

	httputil.JSONResponse(w, http.StatusOK, response)
}
