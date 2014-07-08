package dronboard

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/payment/stripe"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/context"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

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
	return www.SupportedMethodsHandler(&financialsHandler{
		router:    router,
		dataAPI:   dataAPI,
		stripeCli: stripeCli,
	}, []string{"GET", "POST"})
}

func (h *financialsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
			account := context.Get(r, www.CKAccount).(*common.Account)
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
			fmt.Printf("%+v\n", rec)

			if u, err := h.router.Get("doctor-register-success").URLPath(); err != nil {
				www.InternalServerError(w, r, err)
			} else {
				http.Redirect(w, r, u.String(), http.StatusSeeOther)
			}
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
