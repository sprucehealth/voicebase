package dronboard

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/payment/stripe"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type financialsVerifyHandler struct {
	router       *mux.Router
	dataAPI      api.DataAPI
	stripeCli    *stripe.StripeService
	supportEmail string
}

type financialsVerifyForm struct {
	Amount1 string
	Amount2 string

	amount1 int
	amount2 int
}

func (r *financialsVerifyForm) Validate() map[string]string {
	errors := map[string]string{}
	var err error
	if r.Amount1 == "" {
		errors["Amount1"] = "Amount 1 is required"
	} else if r.amount1, err = parseAmount(r.Amount1); err != nil {
		errors["Amount1"] = "Amount 1 is invalid. Please enter a dollar value such as 1.02"
	}
	if r.Amount2 == "" {
		errors["Amount2"] = "Amount 2 is required"
	} else if r.amount2, err = parseAmount(r.Amount2); err != nil {
		errors["Amount2"] = "Amount 2 is invalid. Please enter a dollar value such as 1.02"
	}
	return errors
}

func NewFinancialVerifyHandler(router *mux.Router, dataAPI api.DataAPI, supportEmail string, stripeCli *stripe.StripeService) http.Handler {
	return www.SupportedMethodsHandler(&financialsVerifyHandler{
		router:       router,
		dataAPI:      dataAPI,
		stripeCli:    stripeCli,
		supportEmail: supportEmail,
	}, []string{"GET", "POST"})
}

func (h *financialsVerifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)
	bankAccounts, err := h.dataAPI.ListBankAccounts(account.ID)
	if err != nil {
		www.InternalServerError(w, r, err)
	} else if len(bankAccounts) == 0 {
		if u, err := h.router.Get("doctor-register-financials").URLPath(); err != nil {
			www.InternalServerError(w, r, err)
		} else {
			http.Redirect(w, r, u.String(), http.StatusSeeOther)
		}
		return
	}

	var unverified []*common.BankAccount
	for _, ba := range bankAccounts {
		if !ba.Verified {
			unverified = append(unverified, ba)
		}
	}
	if len(unverified) == 0 {
		h.redirectToNextStep(w, r)
		return
	}

	// TODO: assume for now that there's only one account pending
	toVerify := unverified[0]

	form := &financialsVerifyForm{}
	var errors map[string]string

	var pending, failed, initial bool

	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		if err := schema.NewDecoder().Decode(form, r.PostForm); err != nil {
			www.InternalServerError(w, r, err)
			return
		}

		errors = form.Validate()
		if len(errors) == 0 {
			if (toVerify.VerifyAmount1 == form.amount1 && toVerify.VerifyAmount2 == form.amount2) ||
				(toVerify.VerifyAmount2 == form.amount1 && toVerify.VerifyAmount1 == form.amount2) {
				if err := h.dataAPI.UpdateBankAccountVerficiation(toVerify.ID, 0, 0, "", "", time.Time{}, true); err != nil {
					www.InternalServerError(w, r, err)
					return
				}
				h.redirectToNextStep(w, r)
				return
			}

			errors["Amounts"] = "Amounts do not match the deposits. Please verify everything is entered correctly."
		}
	} else if r.Method == "GET" {
		// On initial page load after creating the account show a different message and
		// don't bother checking the transactions
		fmt.Printf("%+v\n", toVerify.Created)
		if time.Now().UTC().Sub(toVerify.Created) < time.Second*15 {
			initial = true
		} else {
			t1, err := h.stripeCli.GetTransfer(toVerify.VerifyTransfer1ID)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			t2, err := h.stripeCli.GetTransfer(toVerify.VerifyTransfer2ID)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			if t1.Status == "failed" || t2.Status == "failed" {
				failed = true
			} else if t1.Status == "pending" || t2.Status == "pending" {
				pending = true
			}
		}
	}

	www.TemplateResponse(w, http.StatusOK, financialsVerifyTemplate, &www.BaseTemplateContext{
		Title: "Verify Bank Account | Doctor Registration | Spruce",
		SubContext: &financialsVerifyTemplateContext{
			Form:         form,
			FormErrors:   errors,
			SupportEmail: h.supportEmail,
			Pending:      pending,
			Failed:       failed,
			Initial:      initial,
		},
	})
}

func (h *financialsVerifyHandler) redirectToNextStep(w http.ResponseWriter, r *http.Request) {
	if u, err := h.router.Get("doctor-register-success").URLPath(); err != nil {
		www.InternalServerError(w, r, err)
	} else {
		http.Redirect(w, r, u.String(), http.StatusSeeOther)
	}
}
