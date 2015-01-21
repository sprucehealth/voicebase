package promotions

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
)

type routeDoctorPromotion struct {
	promoCodeParams
	DoctorID               int64        `json:"doctor_id"`
	DoctorLongDisplayName  string       `json:"short_display_name"`
	DoctorShortDisplayName string       `json:"long_display_name"`
	DiscountValue          int          `json:"discount_value"`
	DiscountUnit           DiscountUnit `json:"discount_unit"`
}

type context struct {
	DoctorShortDisplayName string
	DoctorLongDisplayName  string
}

func (r *routeDoctorPromotion) Validate() error {

	if err := r.promoCodeParams.Validate(); err != nil {
		return err
	}

	if r.DoctorID == 0 {
		return errors.New("doctor_id required")
	}

	if r.DoctorShortDisplayName == "" {
		return errors.New("short display name of doctor required")
	}

	if r.DoctorLongDisplayName == "" {
		return errors.New("long display name of doctor required")
	}

	ctxt := &context{
		DoctorShortDisplayName: r.DoctorShortDisplayName,
		DoctorLongDisplayName:  r.DoctorLongDisplayName,
	}

	var err error
	r.promoCodeParams.ShortMsg, err = parseMessage(r.promoCodeParams.ShortMsg, ctxt)
	if err != nil {
		return err
	}

	r.promoCodeParams.DisplayMsg, err = parseMessage(r.promoCodeParams.DisplayMsg, ctxt)
	if err != nil {
		return err
	}

	r.promoCodeParams.SuccessMsg, err = parseMessage(r.promoCodeParams.SuccessMsg, ctxt)
	if err != nil {
		return err
	}

	return nil
}

func parseMessage(message string, context interface{}) (string, error) {
	var b bytes.Buffer
	template, err := template.New("test").Parse(message)
	if err != nil {
		return "", err
	}

	if err := template.Execute(&b, context); err != nil {
		return "", err
	}

	return b.String(), nil
}

func (r *routeDoctorPromotion) TypeName() string {
	return routeDoctorType
}

func (r *routeDoctorPromotion) Associate(accountID, codeID int64, expires *time.Time, dataAPI api.DataAPI) error {
	if err := canAssociatePromotionWithAccount(accountID, codeID, r.promoCodeParams.ForNewUser, r.Group(), dataAPI); err != nil {
		return err
	}

	patientID, err := dataAPI.GetPatientIDFromAccountID(accountID)
	if err != nil {
		return err
	}

	// ensure there is no doctor assigned to the patient
	// for the condition that the doctor is signed up to support
	careTeamMembers, err := dataAPI.GetActiveMembersOfCareTeamForPatient(patientID, false)
	if err != nil {
		return err
	}

	// TODO: For now assuming Acne as the pathway. The expected pathway should either be part of the promo
	// or a separate step for allow the patient to select a pathway needs to exist.
	pathway, err := dataAPI.PathwayForTag(api.AcnePathwayTag)
	if err != nil {
		return err
	}

	for _, member := range careTeamMembers {
		if member.PathwayID == pathway.ID &&
			member.ProviderRole == api.DOCTOR_ROLE {
			return &promotionError{
				ErrorMsg: "Code cannot be applied as a doctor already exists in your care team.",
			}
		}
	}

	patientState, err := dataAPI.PatientState(patientID)
	if err != nil {
		return err
	}

	// ensure that the patient can actually be routed to this doctor
	if isEligible, err := dataAPI.DoctorEligibleToTreatInState(patientState, r.DoctorID, pathway.ID); err != nil {
		return err
	} else if !isEligible {
		return &promotionError{
			ErrorMsg: fmt.Sprintf("Code cannot be applied as %s cannot treat patient in %s",
				r.DoctorLongDisplayName, patientState),
		}
	}

	// assign doctor to patient care team
	if err := dataAPI.AddDoctorToCareTeamForPatient(patientID, pathway.ID, r.DoctorID); err != nil {
		return err
	}

	// create pending patient promotion if the discount value > 0,
	// else create completed promotion
	promotionStatus := common.PSPending
	if r.DiscountValue == 0 {
		promotionStatus = common.PSCompleted
	}

	if err := dataAPI.CreateAccountPromotion(&common.AccountPromotion{
		AccountID: accountID,
		Status:    promotionStatus,
		Group:     r.promoCodeParams.PromoGroup,
		CodeID:    codeID,
		Data:      r,
		Expires:   expires,
	}); err != nil {
		return err
	}

	return nil
}

func (r *routeDoctorPromotion) Apply(cost *common.CostBreakdown) (bool, error) {
	if r.DiscountValue > 0 {
		applied, err := applyDiscount(cost, r, r.DiscountUnit, r.DiscountValue)
		if err != nil {
			return false, err
		}

		r.DiscountValue = 0
		return applied, nil
	}

	return false, nil
}

func (r *routeDoctorPromotion) IsConsumed() bool {
	return r.DiscountValue == 0
}

func (r *routeDoctorPromotion) ShortMsg() string {
	return r.promoCodeParams.ShortMsg
}

func (r *routeDoctorPromotion) DisplayMsg() string {
	return r.promoCodeParams.DisplayMsg
}

func (r *routeDoctorPromotion) SuccessMsg() string {
	return r.promoCodeParams.SuccessMsg
}
