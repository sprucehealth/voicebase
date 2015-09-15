package dronboard

import (
	"crypto/rand"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/www"
	"github.com/sprucehealth/schema"
	"golang.org/x/net/context"
)

var verifyDuration = time.Hour * 24 * 7

type financialsHandler struct {
	router    *mux.Router
	dataAPI   api.DataAPI
	stripeCli *stripe.Client
	template  *template.Template
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

func newFinancialsHandler(router *mux.Router, dataAPI api.DataAPI, stripeCli *stripe.Client, templateLoader *www.TemplateLoader) httputil.ContextHandler {
	return httputil.SupportedMethods(&financialsHandler{
		router:    router,
		dataAPI:   dataAPI,
		stripeCli: stripeCli,
		template:  templateLoader.MustLoadTemplate("dronboard/financials.html", "dronboard/base.html", nil),
	}, httputil.Get, httputil.Post)
}

func (h *financialsHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	account := www.MustCtxAccount(ctx)

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
			doctor, err := h.dataAPI.GetDoctorFromAccountID(account.ID)
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
					"doctor_id": strconv.FormatInt(doctor.ID.Int64(), 10),
				},
			}
			rec, err := h.stripeCli.CreateRecipient(rr)
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
			_, err = h.dataAPI.AddBankAccount(&common.BankAccount{
				AccountID:         account.ID,
				StripeRecipientID: rec.ID,
				Default:           true,
				VerifyAmount1:     amount1,
				VerifyTransfer1ID: tx1.ID,
				VerifyAmount2:     amount2,
				VerifyTransfer2ID: tx2.ID,
				VerifyExpires:     expires,
				Verified:          false,
			})
			if err != nil {
				www.InternalServerError(w, r, err)
				return
			}

			h.redirectToNextStep(w, r)
			return
		}
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &www.BaseTemplateContext{
		Title: "Financials | Doctor Registration | Spruce",
		SubContext: &struct {
			Form       *financialsForm
			FormErrors map[string]string
			StripeKey  string
		}{
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
