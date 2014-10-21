package cost

import (
	"encoding/json"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/sku"
)

func totalCostForItems(itemTypes []sku.SKU, patientID int64, updateState bool, dataAPI api.DataAPI, analyticsLogger analytics.Logger) (*common.CostBreakdown, error) {

	costBreakdown := &common.CostBreakdown{}

	for _, itemType := range itemTypes {

		itemCost, err := dataAPI.GetActiveItemCost(itemType)
		if err != nil {
			return nil, err
		}

		costBreakdown.ItemCosts = append(costBreakdown.ItemCosts, itemCost)
		costBreakdown.LineItems = append(costBreakdown.LineItems, itemCost.LineItems...)
	}
	costBreakdown.CalculateTotal()

	if err := applyPromotion(costBreakdown,
		updateState, patientID, dataAPI, analyticsLogger); err != nil {
		return nil, err
	}

	// now apply account credits if there is still a non-zero amount left on the cost
	if err := applyCredits(costBreakdown, patientID, updateState, dataAPI, analyticsLogger); err != nil {
		return nil, err
	}

	return costBreakdown, nil
}

func applyPromotion(costBreakdown *common.CostBreakdown,
	updateState bool, patientID int64, dataAPI api.DataAPI, analyticsLogger analytics.Logger) error {

	// check for any pending promotions
	pendingPromotions, err := dataAPI.PendingPromotionsForPatient(patientID, promotions.Types)
	if err != nil {
		return err
	}

	// TODO: Sort the pending promotions based on the items in the cost breakdown
	// so that we can apply the best promotion applicable to the item

	// apply any promotion associated with the patient account
	for _, pendingPromotion := range pendingPromotions {
		promotion := pendingPromotion.Data.(promotions.Promotion)
		applied, err := promotion.Apply(costBreakdown)
		if err != nil {
			return err
		} else if !applied {
			continue
		}

		if updateState {
			var promotionStatus *common.PromotionStatus
			if promotion.IsConsumed() {
				status := common.PSCompleted
				promotionStatus = &status
			}

			if err := dataAPI.UpdatePatientPromotion(patientID,
				pendingPromotion.CodeID, &api.PatientPromotionUpdate{
					PromotionData: pendingPromotion.Data,
					Status:        promotionStatus,
				}); err != nil {
				return err
			}

			jsonData, err := json.Marshal(map[string]interface{}{
				"code": pendingPromotion.Code,
			})
			if err != nil {
				golog.Errorf(err.Error())
			}

			analyticsLogger.WriteEvents([]analytics.Event{
				&analytics.ServerEvent{
					Event:     "promo_code_consumed",
					Timestamp: analytics.Time(time.Now()),
					PatientID: patientID,
					ExtraJSON: string(jsonData),
				},
			})

		}
	}

	costBreakdown.CalculateTotal()
	return nil
}

func applyCredits(costBreakdown *common.CostBreakdown, patientID int64, updateState bool, dataAPI api.DataAPI, analyticsLogger analytics.Logger) error {
	// now apply account credits if there is still a non-zero amount left on the cost
	if costBreakdown.TotalCost.Amount <= 0 {
		return nil
	}

	patientCredit, err := dataAPI.PatientCredit(patientID)
	if err != api.NoRowsError && err != nil {
		return err
	} else if patientCredit == nil {
		return nil
	}

	creditsToUse := patientCredit.Credit
	if costBreakdown.TotalCost.Amount < creditsToUse {
		creditsToUse = costBreakdown.TotalCost.Amount
	}

	// add line items to the cost breakdown to indicate the amount
	// of spruce credits applied
	costBreakdown.LineItems =
		append(costBreakdown.LineItems, &common.LineItem{
			Description: "Spruce credits",
			Cost: common.Cost{
				Currency: promotions.USDUnit.String(),
				Amount:   -creditsToUse,
			},
		})

	if updateState {
		// update the credits in the account
		if err := dataAPI.UpdateCredit(patientID,
			-creditsToUse, promotions.USDUnit.String()); err != nil {
			return err
		}

		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "credits_consumed",
				Timestamp: analytics.Time(time.Now()),
				PatientID: patientID,
			},
		})
	}

	costBreakdown.CalculateTotal()
	return nil
}
