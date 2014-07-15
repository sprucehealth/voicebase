package dronboard

import (
	"crypto/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/payment/stripe"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

var verifyDuration = time.Hour * 24 * 7

type financialsHandler struct {
	router    *mux.Router
	dataAPI   api.DataAPI
	stripeCli *stripe.StripeService
}

type financialsForm struct {
	StripeToken string
}

func (r *financialsForm) Validate() map[string]string {
	errors := map[string]string{}
	if r.StripeToken == "" {
		errors["StripeToken"] = "Missing token"
	}
	return errors
}

func NewFinancialsHandler(router *mux.Router, dataAPI api.DataAPI, stripeCli *stripe.StripeService) http.Handler {
	return httputil.SupportedMethods(&financialsHandler{
		router:    router,
		dataAPI:   dataAPI,
		stripeCli: stripeCli,
	}, []string{"GET", "POST"})
}

func (h *financialsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, www.CKAccount).(*common.Account)

	// If the doctor already set a bank account then skip this step
	bankAccounts, err := h.dataAPI.ListBankAccounts(account.ID)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	} else if len(bankAccounts) != 0 {
		h.redirectToNextStep(w, r)
		return
	}

	form := &financialsForm{}
	var errors map[string]string

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
			doctor, err := h.dataAPI.GetDoctorFromAccountId(account.ID)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			rr := &stripe.CreateRecipientRequest{
				Name:             doctor.FirstName + " " + doctor.LastName,
				Type:             stripe.Individual,
				Email:            account.Email,
				BankAccountToken: form.StripeToken,
				Metadata: map[string]string{
					"role":      account.Role,
					"doctor_id": strconv.FormatInt(doctor.DoctorId.Int64(), 10),
				},
			}
			rec, err := h.stripeCli.CreateRecipient(rr)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			bankID, err := h.dataAPI.AddBankAccount(account.ID, rec.ID, true)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			// Generate random amounts
			var b [2]byte
			if _, err := rand.Read(b[:]); err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			// Range amount from $0.15 to $1.42
			amount1 := (int(b[0]) / 2) + 15
			amount2 := (int(b[1]) / 2) + 15

			treq := &stripe.CreateTransferRequest{
				Amount:               amount1,
				Currency:             stripe.USD,
				RecipientID:          rec.ID,
				Description:          "verify amount 1",
				StatementDescription: "VERIFY",
			}
			tx1, err := h.stripeCli.CreateTransfer(treq)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}
			treq.Amount = amount2
			treq.Description = "verify amount 2"
			tx2, err := h.stripeCli.CreateTransfer(treq)
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			expires := time.Now().Add(verifyDuration)
			if err := h.dataAPI.UpdateBankAccountVerficiation(bankID, amount1, amount2, tx1.ID, tx2.ID, expires, false); err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			h.redirectToNextStep(w, r)
			return
		}
	}

	www.TemplateResponse(w, http.StatusOK, financialsTemplate, &www.BaseTemplateContext{
		Title: "Financials | Doctor Registration | Spruce",
		SubContext: &financialsTemplateContext{
			Form:       form,
			FormErrors: errors,
			StripeKey:  h.stripeCli.PublishableKey,
		},
	})
}

func (h *financialsHandler) redirectToNextStep(w http.ResponseWriter, r *http.Request) {
	if u, err := h.router.Get("doctor-register-financials-verify").URLPath(); err != nil {
		www.InternalServerError(w, r, err)
	} else {
		http.Redirect(w, r, u.String(), http.StatusSeeOther)
	}
}
