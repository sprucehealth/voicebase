package promotions

import (
	"errors"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
)

type DiscountUnit string

func (d DiscountUnit) String() string {
	return string(d)
}

const (
	PercentUnit DiscountUnit = "%"
	USDUnit     DiscountUnit = "USD"
)

type percentDiscountPromotion struct {
	promoCodeParams
	Type          string `json:"type"`
	DiscountValue int    `json:"value"`
}

type moneyDiscountPromotion struct {
	promoCodeParams
	Type          string `json:"type"`
	DiscountValue int    `json:"value"`
}

func (d *percentDiscountPromotion) Validate() error {
	if err := d.promoCodeParams.Validate(); err != nil {
		return err
	}

	if d.DiscountValue == 0 {
		return errors.New("zero discount value when running a discount only promotion")
	}

	return nil
}

func (d *percentDiscountPromotion) TypeName() string {
	return percentOffType
}

func (d *percentDiscountPromotion) Associate(accountID, codeID int64, expires *time.Time, dataAPI api.DataAPI) error {
	return associate(d, d.promoCodeParams.ForNewUser, accountID, codeID, expires, dataAPI)
}

func (d *percentDiscountPromotion) Apply(cost *common.CostBreakdown) (bool, error) {

	applied, err := applyDiscount(cost, d, PercentUnit, d.DiscountValue)
	if err != nil {
		return false, err
	}

	// Mark the promotion as being used
	d.DiscountValue = 0

	return applied, nil
}

func (d *percentDiscountPromotion) IsConsumed() bool {
	return d.DiscountValue == 0
}

func (d *moneyDiscountPromotion) Validate() error {
	if err := d.promoCodeParams.Validate(); err != nil {
		return err
	}

	if d.DiscountValue == 0 {
		return errors.New("zero discount value when running a discount only promotion")
	}

	return nil
}

func (d *moneyDiscountPromotion) TypeName() string {
	return moneyOffType
}

func (d *moneyDiscountPromotion) Associate(accountID, codeID int64, expires *time.Time, dataAPI api.DataAPI) error {
	return associate(d, d.promoCodeParams.ForNewUser, accountID, codeID, expires, dataAPI)
}

func (d *moneyDiscountPromotion) Apply(cost *common.CostBreakdown) (bool, error) {

	applied, err := applyDiscount(cost, d, USDUnit, d.DiscountValue)
	if err != nil {
		return false, err
	}

	// Mark the promotion as being used
	d.DiscountValue = 0

	return applied, nil
}

func (d *moneyDiscountPromotion) IsConsumed() bool {
	return d.DiscountValue == 0
}

func associate(promotion Promotion, forNewUser bool, accountID, codeID int64, expires *time.Time, dataAPI api.DataAPI) error {
	if err := canAssociatePromotionWithAccount(accountID, codeID, forNewUser,
		promotion.Group(), dataAPI); err != nil {
		return err
	}

	if err := dataAPI.CreateAccountPromotion(&common.AccountPromotion{
		AccountID: accountID,
		Status:    common.PSPending,
		Group:     promotion.Group(),
		CodeID:    codeID,
		Data:      promotion,
		Expires:   expires,
	}); err != nil {
		return err
	}

	return nil
}

func applyDiscount(cost *common.CostBreakdown, promotion Promotion, discountUnit DiscountUnit, discountValue int) (bool, error) {
	// look for the item that belongs to the visit SKU category
	var visitItemCost *common.ItemCost
	for _, item := range cost.ItemCosts {
		if *item.SKUCategory == common.SCVisit {
			visitItemCost = item
			break
		}
	}
	if visitItemCost == nil {
		return false, nil
	}

	// Only Apply to cost if no other promotion has already been applied
	if visitItemCost.PromoApplied {
		return false, nil
	}

	// Only Apply if not already consumed
	if promotion.IsConsumed() {
		return false, nil
	}

	// Only Apply if current total cost is greater than 0
	cost.CalculateTotal()
	if cost.TotalCost.Amount <= 0 {
		return false, nil

	}

	// Calculate discount based on the type and value
	var discount common.Cost
	switch discountUnit {
	case PercentUnit:
		discount = common.Cost{
			Currency: visitItemCost.LineItems[0].Cost.Currency,
			Amount:   -visitItemCost.LineItems[0].Cost.Amount * discountValue / 100,
		}
	default:
		// ensure not to apply a bigger discount value than the cost of the item
		totalCostForVisit := 0
		for _, lineItem := range visitItemCost.LineItems {
			totalCostForVisit += lineItem.Cost.Amount
		}

		if discountValue > totalCostForVisit {
			discountValue = totalCostForVisit
		}

		discount = common.Cost{
			Currency: visitItemCost.LineItems[0].Cost.Currency,
			Amount:   -discountValue,
		}
	}

	//  Create line item and append to cost breakdown
	cost.LineItems = append(cost.LineItems, &common.LineItem{
		Description: promotion.ShortMessage(),
		Cost:        discount,
	})

	// mark that we applied a promotion to the visitItemCost
	visitItemCost.PromoApplied = true

	return true, nil
}
