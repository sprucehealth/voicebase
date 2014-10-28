package promotions

import (
	"errors"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
)

type accountCreditPromotion struct {
	promoCodeParams
	CreditValue int `json:"value"`
}

func (a *accountCreditPromotion) TypeName() string {
	return accountCreditType
}

func (a *accountCreditPromotion) Validate() error {
	if err := a.promoCodeParams.Validate(); err != nil {
		return err
	}

	if a.CreditValue == 0 {
		return errors.New("zero credit value when running an account credit promotion")
	}

	return nil
}

func (a *accountCreditPromotion) Associate(accountID, codeID int64, expires *time.Time, dataAPI api.DataAPI) error {
	if err := canAssociatePromotionWithAccount(accountID, codeID, a.promoCodeParams.ForNewUser,
		a.promoCodeParams.Group(), dataAPI); err != nil {
		return err
	}

	// Add to existing account credits and decrement count to 0
	if err := dataAPI.UpdateCredit(accountID, a.CreditValue, USDUnit.String()); err != nil {
		return err
	}
	a.CreditValue = 0

	if err := dataAPI.CreateAccountPromotion(&common.AccountPromotion{
		AccountID: accountID,
		Status:    common.PSCompleted,
		Group:     a.promoCodeParams.PromoGroup,
		CodeID:    codeID,
		Data:      a,
		Expires:   expires,
	}); err != nil {
		return err
	}

	return nil
}

func (a *accountCreditPromotion) Apply(cost *common.CostBreakdown) (bool, error) {
	// nothing to do since the account credits were consumed at the time of association to the user account
	return false, nil
}

func (a *accountCreditPromotion) IsConsumed() bool {
	return a.CreditValue == 0
}
