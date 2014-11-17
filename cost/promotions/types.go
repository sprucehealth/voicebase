package promotions

import (
	"reflect"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
)

type Promotion interface {
	// Functionality related methods
	TypeName() string
	Validate() error
	Associate(accountID, codeID int64, expires *time.Time, dataAPI api.DataAPI) error
	Apply(costBreakdown *common.CostBreakdown) (bool, error)
	IsConsumed() bool
	Group() string

	// Display related methods
	DisplayMessage() string
	ShortMessage() string
	SuccessMessage() string
	ImageURL() string
}

type ReferralProgram interface {
	TypeName() string
	Title() string
	Description() string
	ShareTextInfo() *ShareTextParams
	Validate() error
	SetOwnerAccountID(accountID int64)
	PromotionForReferredAccount(code string) *common.Promotion
	ReferredAccountAssociatedCode(accountID, codeID int64, dataAPI api.DataAPI) error
	ReferredAccountSubmittedVisit(accountID, codeID int64, dataAPI api.DataAPI) error
	UsersAssociatedCount() int
	VisitsSubmittedCount() int
}

var (
	PromotionOnlyForNewUsersError = &promotionError{ErrorMsg: "This code is only valid for new users"}
	PromotionAlreadyApplied       = &promotionError{ErrorMsg: "This promotion has already been applied to your account"}
	PromotionAlreadyExists        = &promotionError{ErrorMsg: "Promotion already exists"}
	PromotionExpired              = &promotionError{ErrorMsg: "Sorry, promotion code is no longer valid"}
	InvalidCode                   = &promotionError{ErrorMsg: "You entered an invalid promotion code"}
	Types                         = make(map[string]reflect.Type)
)

func NewPercentOffVisitPromotion(percentOffValue int,
	group, displayMsg, shortMsg, successMsg string,
	forNewUser bool) Promotion {
	return &percentDiscountPromotion{
		promoCodeParams: promoCodeParams{
			DisplayMsg: displayMsg,
			SuccessMsg: successMsg,
			ShortMsg:   shortMsg,
			PromoGroup: group,
			ForNewUser: forNewUser,
		},
		DiscountValue: percentOffValue,
	}
}

func NewMoneyOffVisitPromotion(discountValue int,
	group, displayMsg, shortMsg, successMsg string,
	forNewUser bool) Promotion {
	return &moneyDiscountPromotion{
		promoCodeParams: promoCodeParams{
			DisplayMsg: displayMsg,
			SuccessMsg: successMsg,
			ShortMsg:   shortMsg,
			PromoGroup: group,
			ForNewUser: forNewUser,
		},
		DiscountValue: discountValue,
	}
}

func NewAccountCreditPromotion(creditValue int, group, displayMsg, shortMsg, successMsg string,
	forNewUser bool) Promotion {
	return &accountCreditPromotion{
		promoCodeParams: promoCodeParams{
			DisplayMsg: displayMsg,
			SuccessMsg: successMsg,
			ShortMsg:   shortMsg,
			PromoGroup: group,
			ForNewUser: forNewUser,
		},
		CreditValue: creditValue,
	}
}

func NewRouteDoctorPromotion(doctorID int64,
	doctorLongDisplayName,
	doctorShortDisplayName,
	smallThumbnailURL string,
	group, displayMsg, shortMsg,
	successMsg string,
	discountValue int,
	discountUnit DiscountUnit) (Promotion, error) {

	rd := &routeDoctorPromotion{
		promoCodeParams: promoCodeParams{
			DisplayMsg: displayMsg,
			ImgURL:     smallThumbnailURL,
			SuccessMsg: successMsg,
			ShortMsg:   shortMsg,
			PromoGroup: group,
			ForNewUser: true,
		},
		DoctorID:               doctorID,
		DiscountValue:          discountValue,
		DiscountUnit:           discountUnit,
		DoctorLongDisplayName:  doctorLongDisplayName,
		DoctorShortDisplayName: doctorShortDisplayName,
	}

	if err := rd.Validate(); err != nil {
		return nil, err
	}

	return rd, nil
}
