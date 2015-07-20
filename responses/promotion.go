package responses

import "github.com/sprucehealth/backend/common"

// ClientPromotion represents the data neede dy a client to correctly display a promotion
type ClientPromotion struct {
	Code                 string `json:"display_code"`
	Description          string `json:"description"`
	DescriptionHasTokens bool   `json:"description_has_tokens"`
	ExpirationDate       int64  `json:"expiration_date"`
}

// Promotion represents a promotion within the system
type Promotion struct {
	Code    string       `json:"code"`
	CodeID  int64        `json:"code_id,string"`
	Data    common.Typed `json:"data"`
	Type    string       `json:"type"`
	Group   string       `json:"group"`
	Expires *int64       `json:"expires"`
	Created int64        `json:"created"`
}

// TransformPromotion transforms the provided data record to the corresponding response format
func TransformPromotion(p *common.Promotion) *Promotion {
	promo := &Promotion{
		Code:    p.Code,
		CodeID:  p.CodeID,
		Data:    p.Data,
		Type:    p.Data.TypeName(),
		Group:   p.Group,
		Created: p.Created.Unix(),
	}
	if p.Expires != nil {
		t := p.Expires.Unix()
		promo.Expires = &t
	}
	return promo
}

// PromotionReferralRoute represents a route to a promotion within the system
type PromotionReferralRoute struct {
	ID              int64               `json:"id,string"`
	PromotionCodeID int64               `json:"promotion_code_id,string"`
	Created         int64               `json:"created"`
	Modified        int64               `json:"modified"`
	Priority        int                 `json:"priority"`
	Lifecycle       common.PRRLifecycle `json:"lifecycle"`
	Gender          *common.PRRGender   `json:"gender"`
	AgeLower        *int                `json:"age_lower"`
	AgeUpper        *int                `json:"age_upper"`
	State           *string             `json:"state"`
	Pharmacy        *string             `json:"pharmacy"`
}

// TransformPromotionReferralRoute transforms the provided data record to the corresponding response format
func TransformPromotionReferralRoute(r *common.PromotionReferralRoute) *PromotionReferralRoute {
	return &PromotionReferralRoute{
		ID:              r.ID,
		PromotionCodeID: r.PromotionCodeID,
		Created:         r.Created.Unix(),
		Modified:        r.Modified.Unix(),
		Priority:        r.Priority,
		Lifecycle:       r.Lifecycle,
		Gender:          r.Gender,
		AgeLower:        r.AgeLower,
		AgeUpper:        r.AgeUpper,
		State:           r.State,
		Pharmacy:        r.Pharmacy,
	}
}

// ReferralProgramTemplate represents a referral program template within the system
type ReferralProgramTemplate struct {
	ID              int64                        `json:"id,string"`
	Role            string                       `json:"role"`
	RoleTypeID      int64                        `json:"role_type_id,string"`
	Data            common.Typed                 `json:"data"`
	Created         int64                        `json:"created"`
	Status          common.ReferralProgramStatus `json:"status"`
	PromotionCodeID *int64                       `json:"promotion_code_id,string"`
}

// TransformReferralProgramTemplate transforms the provided data record to the corresponding response format
func TransformReferralProgramTemplate(r *common.ReferralProgramTemplate) *ReferralProgramTemplate {
	return &ReferralProgramTemplate{
		ID:              r.ID,
		Role:            r.Role,
		RoleTypeID:      r.RoleTypeID,
		Data:            r.Data,
		Created:         r.Created.Unix(),
		Status:          r.Status,
		PromotionCodeID: r.PromotionCodeID,
	}
}

// PromotionGroup represents a promotion group within the system
type PromotionGroup struct {
	ID               int64  `json:"id,string"`
	Name             string `json:"name"`
	MaxAllowedPromos int    `json:"max_allowed_promos"`
}

// TransformPromotionGroup transforms the provided data record to the corresponding response format
func TransformPromotionGroup(pg *common.PromotionGroup) *PromotionGroup {
	return &PromotionGroup{
		ID:               pg.ID,
		Name:             pg.Name,
		MaxAllowedPromos: pg.MaxAllowedPromos,
	}
}
