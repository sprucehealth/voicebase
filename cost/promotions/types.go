package promotions

import (
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

const (
	// DefaultPromotionImageURL represents the fallback URL to use for legacy promotions or promotions that did not provide an image URL
	DefaultPromotionImageURL = "https://d2bln09x7zhlg8.cloudfront.net/icon_share_default_160_x_160.png"

	// DefaultPromotionImageWidth represents the fallback Width to use in association with DefaultPromotionImageURL
	DefaultPromotionImageWidth = 80

	// DefaultPromotionImageHeight represents the fallback Height to use in association with DefaultPromotionImageURL
	DefaultPromotionImageHeight = 80
)

// Promotion is an interface that is intended to capture all the functionality required by the system to generically interact with and service requests related to promotions
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
	ImageWidth() int
	ImageHeight() int
	IsZeroValue() bool
}

// ReferralProgram is an interface that is intended to capture all the functionality required by the system to generically interact with and service requests related to referral programs
type ReferralProgram interface {
	HomeCardText() string
	HomeCardImageURL() *app_url.SpruceAsset
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
	// ErrPromotionOnlyForNewUsers should be returned when a promo is being applied that is intended only for new users to a non new user account
	ErrPromotionOnlyForNewUsers = &promotionError{ErrorMsg: "This code is only valid for new users"}

	// ErrPromotionAlreadyApplied should be returned when a promo is being applied to an account that has previously applied the promotion
	ErrPromotionAlreadyApplied = &promotionError{ErrorMsg: "This promotion has already been applied to your account"}

	// ErrPromotionTypeMaxClaimed should be returned when a promo is being applied to an account that has already claimed the max ammount allowed by this promotion group
	ErrPromotionTypeMaxClaimed = &promotionError{ErrorMsg: "The limit on this promotion type has already been reached by your account"}

	// ErrPromotionExpired should be returned when a promo is being applied that has expired
	ErrPromotionExpired = &promotionError{ErrorMsg: "Sorry, promotion code is no longer valid"}

	// ErrInvalidCode should be returned when a promo code is being applied that doesn't map to a valid promotion
	ErrInvalidCode = &promotionError{ErrorMsg: "You entered an invalid promotion code"}
)

// NewPercentOffVisitPromotion returns a new initialized instance of percentDiscountPromotion
func NewPercentOffVisitPromotion(percentOffValue int,
	group, displayMsg, shortMsg, successMsg, imageURL string,
	imageWidth, ImageHeight int,
	forNewUser bool) Promotion {
	return &percentDiscountPromotion{
		promoCodeParams: promoCodeParams{
			DisplayMsg: displayMsg,
			SuccessMsg: successMsg,
			ShortMsg:   shortMsg,
			PromoGroup: group,
			ForNewUser: forNewUser,
			ImgURL:     imageURL,
			ImgWidth:   imageWidth,
			ImgHeight:  ImageHeight,
		},
		DiscountValue: percentOffValue,
	}
}

// NewMoneyOffVisitPromotion returns a new initialized instance of moneyDiscountPromotion
func NewMoneyOffVisitPromotion(discountValue int,
	group, displayMsg, shortMsg, successMsg, imageURL string,
	imageWidth, ImageHeight int,
	forNewUser bool) Promotion {
	return &moneyDiscountPromotion{
		promoCodeParams: promoCodeParams{
			DisplayMsg: displayMsg,
			SuccessMsg: successMsg,
			ShortMsg:   shortMsg,
			PromoGroup: group,
			ForNewUser: forNewUser,
			ImgURL:     imageURL,
			ImgWidth:   imageWidth,
			ImgHeight:  ImageHeight,
		},
		DiscountValue: discountValue,
	}
}

// NewAccountCreditPromotion returns a new initialized instance of accountCreditPromotion
func NewAccountCreditPromotion(creditValue int, group, displayMsg, shortMsg, successMsg, imageURL string,
	imageWidth, ImageHeight int,
	forNewUser bool) Promotion {
	return &accountCreditPromotion{
		promoCodeParams: promoCodeParams{
			DisplayMsg: displayMsg,
			SuccessMsg: successMsg,
			ShortMsg:   shortMsg,
			PromoGroup: group,
			ForNewUser: forNewUser,
			ImgURL:     imageURL,
			ImgWidth:   imageWidth,
			ImgHeight:  ImageHeight,
		},
		CreditValue: creditValue,
	}
}

// NewRouteDoctorPromotion returns a new initialized instance of routeDoctorPromotion
func NewRouteDoctorPromotion(doctorID int64,
	doctorLongDisplayName,
	doctorShortDisplayName,
	smallThumbnailURL,
	group, displayMsg, shortMsg,
	successMsg string,
	discountValue int,
	discountUnit DiscountUnit) (Promotion, error) {

	rd := &routeDoctorPromotion{
		promoCodeParams: promoCodeParams{
			DisplayMsg: displayMsg,
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
