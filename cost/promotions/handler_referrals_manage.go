package promotions

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type referralProgramTemplateHandler struct {
	dataAPI api.DataAPI
}

type referralProgramsTemplateRequestData struct {
	Promotion   json.RawMessage `json:"promotion"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	ShareText   string          `json:"share_text"`
	Group       string          `json:"group"`
}

func NewReferralProgramTemplateHandler(dataAPI api.DataAPI) http.Handler {
	return &referralProgramTemplateHandler{
		dataAPI: dataAPI,
	}
}

func (p *referralProgramTemplateHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.ADMIN_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	if r.Method != apiservice.HTTP_POST {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (p *referralProgramTemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rd referralProgramsTemplateRequestData
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	promotionData := &moneyDiscountPromotion{}
	if err := json.Unmarshal(rd.Promotion, &promotionData); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	if err := promotionData.Validate(); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	// currently we only support the give program
	referralProgram := NewGiveReferralProgram(rd.Title, rd.Description,
		rd.Group, promotionData)

	referralProgramTemplate := &common.ReferralProgramTemplate{
		Role:   api.PATIENT_ROLE,
		Data:   referralProgram,
		Status: common.RSActive,
	}

	if _, err := p.dataAPI.CreateReferralProgramTemplate(referralProgramTemplate); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, referralProgramTemplate)
}
