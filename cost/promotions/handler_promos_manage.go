package promotions

import (
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
)

type promotionHandler struct {
	dataAPI api.DataAPI
}

type managePromotionsRequestData struct {
	Type      string          `json:"type"`
	Promotion json.RawMessage `json:"promotion"`
	Expires   string          `json:"expires"`
	Code      string          `json:"code"`
}

func NewPromotionsHandler(dataAPI api.DataAPI) http.Handler {
	return &promotionHandler{
		dataAPI: dataAPI,
	}
}

func (p *promotionHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.ADMIN_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	switch r.Method {
	case apiservice.HTTP_POST:
	default:
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (p *promotionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_POST:
		p.addPromotion(w, r)
	}
}

func (p *promotionHandler) addPromotion(w http.ResponseWriter, r *http.Request) {
	var rd managePromotionsRequestData
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	promotionDataType, ok := Types[rd.Type]
	if !ok {
		apiservice.WriteValidationError("Unknown type "+rd.Type, w, r)
		return
	}

	promotionData := reflect.New(promotionDataType).Interface().(Promotion)
	if err := json.Unmarshal(rd.Promotion, &promotionData); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	// validate promotion
	if err := promotionData.Validate(); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	promoCode := rd.Code
	var err error
	if promoCode != "" {
		_, err = p.dataAPI.LookupPromoCode(promoCode)
		if err != nil && err != api.NoRowsError {
			apiservice.WriteError(err, w, r)
			return
		} else if err == nil {
			apiservice.WriteValidationError("Promotion with this code already exists", w, r)
			return
		}
	} else {
		promoCode, err = GeneratePromoCode(p.dataAPI)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	var expiration *time.Time
	if rd.Expires != "" {
		exp, err := time.Parse("2006-01-02", rd.Expires)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		expiration = &exp
	}

	promotion := &common.Promotion{
		Code:    promoCode,
		Data:    promotionData,
		Group:   promotionData.Group(),
		Expires: expiration,
	}

	if err := p.dataAPI.CreatePromotion(promotion); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, promotion)
}
