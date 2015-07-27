package promotions

import (
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

type giveReferralProgram struct {
	referralProgramParams
	Group           string `json:"group"`
	AssociatedCount int    `json:"associated_count"`
	SubmittedCount  int    `json:"visit_submitted_count"`
}

func (g *giveReferralProgram) TypeName() string {
	return giveReferralType
}

func (g *giveReferralProgram) HomeCardText() string {
	if g.referralProgramParams.HomeCard == nil {
		return ""
	}
	return g.referralProgramParams.HomeCard.Text
}

func (g *giveReferralProgram) HomeCardImageURL() *app_url.SpruceAsset {
	if g.referralProgramParams.HomeCard == nil {
		return app_url.IconPromoLogo
	}
	return g.referralProgramParams.HomeCard.ImageURL
}

func (g *giveReferralProgram) Title() string {
	return g.referralProgramParams.Title
}

func (g *giveReferralProgram) Description() string {
	return g.referralProgramParams.Description
}

func (g *giveReferralProgram) ShareTextInfo() *ShareTextParams {
	return g.referralProgramParams.ShareText
}

func (g *giveReferralProgram) SetOwnerAccountID(accountID int64) {
	g.OwnerAccountID = accountID
}

func (g *giveReferralProgram) UsersAssociatedCount() int {
	return g.AssociatedCount
}

func (g *giveReferralProgram) VisitsSubmittedCount() int {
	return g.SubmittedCount
}

type giveMoneyOffReferralProgram struct {
	giveReferralProgram
	Promotion *moneyDiscountPromotion `json:"promotion"`
}

func (g *giveMoneyOffReferralProgram) PromotionForReferredAccount(code string) *common.Promotion {
	return &common.Promotion{
		Code:  code,
		Group: g.Group,
		Data:  g.Promotion,
	}
}

func (g *giveMoneyOffReferralProgram) Validate() error {
	if err := g.referralProgramParams.Validate(); err != nil {
		return err
	}

	if g.Group == "" {
		return errors.New("missing group")
	}

	if g.Promotion == nil {
		return errors.New("missing promotion on referral")
	}

	if err := g.Promotion.Validate(); err != nil {
		return err
	}

	return nil
}

func (g *giveMoneyOffReferralProgram) TypeName() string {
	return giveReferralMoneyOffType
}

func (g *giveMoneyOffReferralProgram) ReferredAccountAssociatedCode(accountID, codeID int64, dataAPI api.DataAPI) error {
	g.AssociatedCount++
	if err := dataAPI.UpdateReferralProgram(g.referralProgramParams.OwnerAccountID, codeID, g); err != nil {
		return err
	}

	if err := dataAPI.TrackAccountReferral(&common.ReferralTrackingEntry{
		CodeID:             codeID,
		ClaimingAccountID:  accountID,
		ReferringAccountID: g.referralProgramParams.OwnerAccountID,
		Status:             common.RTSPending,
	}); err != nil {
		return err
	}

	return nil
}

func (g *giveMoneyOffReferralProgram) ReferredAccountSubmittedVisit(accountID, codeID int64, dataAPI api.DataAPI) error {
	g.SubmittedCount++
	if err := dataAPI.UpdateReferralProgram(g.referralProgramParams.OwnerAccountID, codeID, g); err != nil {
		return err
	}

	if err := dataAPI.UpdateAccountReferral(accountID, common.RTSCompleted); err != nil {
		return err
	}

	return nil
}

type givePercentOffReferralProgram struct {
	giveReferralProgram
	Promotion *percentDiscountPromotion `json:"promotion"`
}

func (g *givePercentOffReferralProgram) PromotionForReferredAccount(code string) *common.Promotion {
	return &common.Promotion{
		Code:  code,
		Group: g.Group,
		Data:  g.Promotion,
	}
}

func (g *givePercentOffReferralProgram) Validate() error {
	if err := g.referralProgramParams.Validate(); err != nil {
		return err
	}

	if g.Group == "" {
		return errors.New("missing group")
	}

	if g.Promotion == nil {
		return errors.New("missing promotion on referral")
	}

	if err := g.Promotion.Validate(); err != nil {
		return err
	}

	return nil
}

func (g *givePercentOffReferralProgram) TypeName() string {
	return giveReferralPercentOffType
}

func (g *givePercentOffReferralProgram) ReferredAccountAssociatedCode(accountID, codeID int64, dataAPI api.DataAPI) error {
	g.AssociatedCount++
	if err := dataAPI.UpdateReferralProgram(g.referralProgramParams.OwnerAccountID, codeID, g); err != nil {
		return err
	}

	if err := dataAPI.TrackAccountReferral(&common.ReferralTrackingEntry{
		CodeID:             codeID,
		ClaimingAccountID:  accountID,
		ReferringAccountID: g.referralProgramParams.OwnerAccountID,
		Status:             common.RTSPending,
	}); err != nil {
		return err
	}

	return nil
}

func (g *givePercentOffReferralProgram) ReferredAccountSubmittedVisit(accountID, codeID int64, dataAPI api.DataAPI) error {
	g.SubmittedCount++
	if err := dataAPI.UpdateReferralProgram(g.referralProgramParams.OwnerAccountID, codeID, g); err != nil {
		return err
	}

	if err := dataAPI.UpdateAccountReferral(accountID, common.RTSCompleted); err != nil {
		return err
	}

	return nil
}

// NewGiveReferralProgram returns a new initialized instance of a ReferralProgram. The type of referral program generated is based off the internal data of the provided Promotion. e.g: (percentOffType -> givePercentOffReferralProgram, moneyOffType -> giveMoneyOffReferralProgram)
func NewGiveReferralProgram(title, description, group string, homeCard *HomeCardConfig, promotion Promotion, shareTextParams *ShareTextParams, imageURL string, imageWidth, imageHeight int) (ReferralProgram, error) {
	grp := giveReferralProgram{
		referralProgramParams: referralProgramParams{
			Title:       title,
			Description: description,
			ImgURL:      imageURL,
			ImgWidth:    imageWidth,
			ImgHeight:   imageHeight,
			ShareText:   shareTextParams,
			HomeCard:    homeCard,
		},
		Group: group,
	}
	switch promotion.TypeName() {
	case percentOffType:
		pdp, ok := promotion.(*percentDiscountPromotion)
		if !ok {
			return nil, errors.New("Unable to cast promotion data as percentDiscountPromotion")
		}
		return &givePercentOffReferralProgram{
			giveReferralProgram: grp,
			Promotion:           pdp,
		}, nil
	case moneyOffType:
		mdp, ok := promotion.(*moneyDiscountPromotion)
		if !ok {
			return nil, errors.New("Unable to cast promotion data as moneyDiscountPromotion")
		}
		return &giveMoneyOffReferralProgram{
			giveReferralProgram: grp,
			Promotion:           mdp,
		}, nil
	}
	return nil, fmt.Errorf("Unknown promotion type for referralGive %q", promotion.TypeName())
}
