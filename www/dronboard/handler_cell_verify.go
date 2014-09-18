package dronboard

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

const (
	smsVerifyExpiration   = 10 * 60
	smsVerifyTokenPurpose = "sms-verify"
)

type cellVerifyHandler struct {
	router     *mux.Router
	dataAPI    api.DataAPI
	authAPI    api.AuthAPI
	smsAPI     api.SMSAPI
	fromNumber string
	template   *template.Template
	nextStep   string
}

type cellVerifyRequest struct {
	Number string `json:"number"`
	Code   string `json:"code"`
}

func NewCellVerifyHandler(router *mux.Router, dataAPI api.DataAPI, authAPI api.AuthAPI, smsAPI api.SMSAPI, fromNumber string, templateLoader *www.TemplateLoader) http.Handler {
	return httputil.SupportedMethods(&cellVerifyHandler{
		router:     router,
		dataAPI:    dataAPI,
		authAPI:    authAPI,
		smsAPI:     smsAPI,
		fromNumber: fromNumber,
		template:   templateLoader.MustLoadTemplate("dronboard/cell_verify.html", "dronboard/base.html", nil),
		nextStep:   "doctor-register-saved-message",
	}, []string{"GET", "POST"})
}

func (h *cellVerifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)

	numbers, err := h.authAPI.GetPhoneNumbersForAccount(account.ID)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	var cell common.Phone
	for _, n := range numbers {
		if n.Status == api.STATUS_ACTIVE && n.Type == api.PHONE_CELL {
			cell = n.Phone
			if n.Verified {
				// A cell number if already verified so go to the next step
				h.navigateNext(w, r)
				return
			}
			break
		}
	}

	if r.Method == "POST" {
		var req cellVerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			www.APIBadRequestError(w, r, "Failed to decode JSON")
			return
		}

		if req.Code != "" {
			acc, err := h.authAPI.ValidateTempToken(smsVerifyTokenPurpose, smsCodeToken(account.ID, cell.String(), req.Code))
			if err == api.TokenDoesNotExist || err == api.TokenExpired {
				www.JSONResponse(w, r, http.StatusForbidden, &www.APIErrorResponse{
					Error: www.APIError{
						Message: "Invalid verification code. Check that it is entered correctly, or try sending a new code.",
					},
				})
				return
			} else if err != nil {
				www.APIInternalError(w, r, err)
				return
			} else if acc.ID != account.ID {
				www.APIInternalError(w, r, errors.New("Account numbers don't match during phone number verification"))
				return
			}

			if err := h.authAPI.ReplacePhoneNumbersForAccount(account.ID, []*common.PhoneNumber{
				&common.PhoneNumber{
					Phone:    cell,
					Type:     api.PHONE_CELL,
					Status:   api.STATUS_ACTIVE,
					Verified: true,
				},
			}); err != nil {
				www.APIInternalError(w, r, err)
			}

			www.JSONResponse(w, r, http.StatusOK, true)
			return
		}

		if req.Number == "" {
			www.APIBadRequestError(w, r, "Number or code missing")
			return
		}

		phone, err := common.ParsePhone(req.Number)
		if err != nil {
			www.JSONResponse(w, r, http.StatusBadRequest, &www.APIErrorResponse{
				Error: www.APIError{
					Message: err.Error(),
				},
			})
			return
		}

		if err := h.authAPI.ReplacePhoneNumbersForAccount(account.ID, []*common.PhoneNumber{
			&common.PhoneNumber{
				Phone:    phone,
				Type:     api.PHONE_CELL,
				Status:   api.STATUS_ACTIVE,
				Verified: false,
			},
		}); err != nil {
			www.APIInternalError(w, r, err)
		}

		code, err := common.GenerateSMSCode()
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		if _, err := h.authAPI.CreateTempToken(account.ID, smsVerifyExpiration, smsVerifyTokenPurpose, smsCodeToken(account.ID, phone.String(), code)); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		if err := h.smsAPI.Send(h.fromNumber, phone.String(), "Your Spruce verification code is "+code); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		www.JSONResponse(w, r, http.StatusOK, true)
		return
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Verify Cell Phone | Doctor Registration | Spruce",
		SubContext: &struct {
			Number  string `json:"number"`
			NextURL string `json:"nextURL"`
		}{
			Number:  cell.String(),
			NextURL: h.nextURL(),
		},
	})
}

func (h *cellVerifyHandler) nextURL() string {
	u, err := h.router.Get(h.nextStep).URLPath()
	if err != nil {
		// Shouldn't happen. If it does it means it's a code issue and can't be handled.
		panic(err)
	}
	return u.String()
}

func (h *cellVerifyHandler) navigateNext(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.nextURL(), http.StatusSeeOther)
}

func smsCodeToken(accoundID int64, number, code string) string {
	return fmt.Sprintf("%d:%s:%s", accoundID, number, code)
}
