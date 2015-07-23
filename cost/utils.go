package cost

import (
	"encoding/json"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/cost/promotions"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/golog"
)

var (
	// BuzzLaunchDate represents the date on which we launched the buzz lightyear iteration of the product
	BuzzLaunchDate = time.Date(2015, time.March, 31, 0, 0, 0, 0, time.UTC)
)

// GlobalFirstVisitFreeEnabled is a Server configurable flag for determining if the first visit for all patients should be free
var GlobalFirstVisitFreeEnabled = &cfg.ValueDef{
	Name:        "Global.First.Visit.Free.Enabled",
	Description: "Enable or disable the first visit free promotion for all patients and visit types.",
	Type:        cfg.ValueTypeBool,
	Default:     true,
}

func totalCostForItems(
	skuTypes []string,
	accountID int64,
	updateState bool,
	dataAPI api.DataAPI,
	analyticsLogger analytics.Logger,
	cfgStore cfg.Store,
) (*common.CostBreakdown, error) {

	costBreakdown := &common.CostBreakdown{}
	for _, skuType := range skuTypes {
		itemCost, err := dataAPI.GetActiveItemCost(skuType)
		if err != nil {
			return nil, err
		}

		costBreakdown.ItemCosts = append(costBreakdown.ItemCosts, itemCost)
		costBreakdown.LineItems = append(costBreakdown.LineItems, itemCost.LineItems...)
	}
	costBreakdown.CalculateTotal()

	if cfgStore.Snapshot().Bool(GlobalFirstVisitFreeEnabled.Name) {
		patientID, err := dataAPI.GetPatientIDFromAccountID(accountID)
		if err != nil {
			return nil, err
		}

		visits, err := dataAPI.VisitsSubmittedForPatientSince(patientID, BuzzLaunchDate)
		if err != nil {
			return nil, err
		}

		switch {
		case len(visits) == 0:
			// apply launch promotion special for the first visit patient is about to submit
			addLaunchPromoToCost(costBreakdown)
			return costBreakdown, nil
		case len(visits) == 1 && visits[0].Status == common.PVStatusSubmitted && updateState:
			// apply launch promotion special when we are committing the cost for the first visit the patient submitted
			addLaunchPromoToCost(costBreakdown)
			return costBreakdown, nil
		}
	}

	if err := applyPromotions(costBreakdown, updateState, accountID, dataAPI, analyticsLogger); err != nil {
		return nil, err
	}

	// now apply account credits if there is still a non-zero amount left on the cost
	if err := applyCredits(costBreakdown, accountID, updateState, dataAPI, analyticsLogger); err != nil {
		return nil, err
	}

	return costBreakdown, nil
}

func addLaunchPromoToCost(costBreakdown *common.CostBreakdown) {
	if costBreakdown.TotalCost.Amount > 0 {
		costBreakdown.LineItems = append(costBreakdown.LineItems, &common.LineItem{
			Description: "Free Visit",
			Cost: common.Cost{
				Currency: promotions.USDUnit.String(),
				Amount:   -costBreakdown.TotalCost.Amount,
			},
		})
		costBreakdown.CalculateTotal()
	}
}

func applyPromotions(costBreakdown *common.CostBreakdown, updateState bool, accountID int64, dataAPI api.DataAPI, analyticsLogger analytics.Logger) error {
	// check for any pending promotions
	pendingPromotions, err := dataAPI.PendingPromotionsForAccount(accountID, common.PromotionTypes)
	if err != nil {
		return err
	}

	// apply any promotions associated with the patient account
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

			if err := dataAPI.UpdateAccountPromotion(accountID,
				pendingPromotion.CodeID, &api.AccountPromotionUpdate{
					PromotionData: pendingPromotion.Data,
					Status:        promotionStatus,
				}, api.APOPendingOnly); err != nil {
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
					AccountID: accountID,
					ExtraJSON: string(jsonData),
				},
			})
		}
	}

	costBreakdown.CalculateTotal()
	return nil
}

func applyCredits(costBreakdown *common.CostBreakdown, accountID int64, updateState bool, dataAPI api.DataAPI, analyticsLogger analytics.Logger) error {
	// now apply account credits if there is still a non-zero amount left on the cost
	if costBreakdown.TotalCost.Amount <= 0 {
		return nil
	}

	accountCredit, err := dataAPI.AccountCredit(accountID)
	if err != nil && !api.IsErrNotFound(err) {
		return err
	} else if accountCredit == nil {
		return nil
	} else if accountCredit.Credit == 0 {
		return nil
	}

	creditsToUse := accountCredit.Credit
	if costBreakdown.TotalCost.Amount < creditsToUse {
		creditsToUse = costBreakdown.TotalCost.Amount
	}

	// add line items to the cost breakdown to indicate the amount
	// of spruce credits applied
	costBreakdown.LineItems =
		append(costBreakdown.LineItems, &common.LineItem{
			Description: "Credits",
			Cost: common.Cost{
				Currency: promotions.USDUnit.String(),
				Amount:   -creditsToUse,
			},
		})

	if updateState {
		// update the credits in the account
		if err := dataAPI.UpdateCredit(accountID,
			-creditsToUse, promotions.USDUnit.String()); err != nil {
			return err
		}

		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "credits_consumed",
				Timestamp: analytics.Time(time.Now()),
				AccountID: accountID,
			},
		})
	}

	costBreakdown.CalculateTotal()
	return nil
}
