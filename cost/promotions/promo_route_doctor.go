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

func (r *routeDoctorPromotion) IsZeroValue() bool {
	return false
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

	// NOTE: at this point we assume a case has been created
	// for the patient such that the doctor can be assigned to the patient's
	// newly created case. It is also assumed that only a single case exists
	// for the patient so that we know which case to assigned the doctor to

	cases, err := dataAPI.GetCasesForPatient(patientID, []string{common.PCStatusOpen.String()})
	if err != nil {
		return err
	} else if len(cases) != 1 {
		return fmt.Errorf("Expected 1 case for the patient instead got %d", len(cases))
	}

	// get the care team for the case
	members, err := dataAPI.GetActiveMembersOfCareTeamForCase(cases[0].ID.Int64(), false)
	if err != nil {
		return err
	}

	for _, member := range members {
		if member.ProviderRole == api.RoleDoctor && member.Status == api.StatusActive {
			return &promotionError{
				ErrorMsg: "Code cannot be applied as a doctor already exists in your care team.",
			}
		}
	}

	// assign doctor to patient care team
	if err := dataAPI.AddDoctorToPatientCase(r.DoctorID, cases[0].ID.Int64()); err != nil {
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
