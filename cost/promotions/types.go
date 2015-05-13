package promotions

import (
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

const (
	DefaultPromotionImageURL    string = "https://d2bln09x7zhlg8.cloudfront.net/icon_share_default_160_x_160.png"
	DefaultPromotionImageWidth  int    = 80
	DefaultPromotionImageHeight int    = 80
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
	ImageWidth() int
	ImageHeight() int
}

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
	ErrPromotionOnlyForNewUsers = &promotionError{ErrorMsg: "This code is only valid for new users"}
	ErrPromotionAlreadyApplied  = &promotionError{ErrorMsg: "This promotion has already been applied to your account"}
	ErrPromotionAlreadyExists   = &promotionError{ErrorMsg: "Promotion already exists"}
	ErrPromotionExpired         = &promotionError{ErrorMsg: "Sorry, promotion code is no longer valid"}
	ErrInvalidCode              = &promotionError{ErrorMsg: "You entered an invalid promotion code"}
)

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
