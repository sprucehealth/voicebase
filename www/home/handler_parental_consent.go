package home

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/www"
)

type parentalConsentHandler struct {
	dataAPI    api.DataAPI
	mediaStore *media.Store
	template   *template.Template
}

type parentalConsentContext struct {
	Account     *common.Account
	Environment string
	Hydration   *parentalConsentHydration
}

type parentalConsentHydration struct {
	ChildDetails               *patientContext
	ParentalConsent            *consentContext
	IsParentSignedIn           bool
	IdentityVerificationImages *identitiyImageContext
}

type patientContext struct {
	PatientID string `json:"patientID"`
	FirstName string `json:"firstName"`
	Gender    string `json:"gender"`
}

type consentContext struct {
	Consented    bool   `json:"consented"`
	Relationship string `json:"relationship"`
}

type identitiyImageContext struct {
	Types map[string]string `json:"types"`
}

func newParentalConsentHandler(dataAPI api.DataAPI, mediaStore *media.Store, templateLoader *www.TemplateLoader) httputil.ContextHandler {
	return httputil.ContextSupportedMethods(&parentalConsentHandler{
		dataAPI:    dataAPI,
		mediaStore: mediaStore,
		template:   templateLoader.MustLoadTemplate("home/parental-consent.html", "", nil),
	}, httputil.Get)
}

func (h *parentalConsentHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// The person may not be signed in which is fine. Account will be nil then.
	account, _ := www.CtxAccount(ctx)

	childPatientID, err := strconv.ParseInt(mux.Vars(ctx)["childid"], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	token := r.FormValue("t")
	if token != "" {
		cookie := newParentalConsentCookie(childPatientID, token, r)
		http.SetCookie(w, cookie)
	} else {
		token = parentalConsentCookie(childPatientID, r)
	}

	var hasAccess bool
	if token != "" {
		hasAccess = patient.ValidateParentalConsentToken(h.dataAPI, token, childPatientID)
	}

	var consent *common.ParentalConsent
	var parentPatientID int64
	idProof := map[string]string{}
	if account != nil {
		parentPatientID, err = h.dataAPI.GetPatientIDFromAccountID(account.ID)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		consents, err := h.dataAPI.ParentalConsent(childPatientID)
		if err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		for _, c := range consents {
			if c.ParentPatientID == parentPatientID {
				consent = c
				break
			}
		}
		if len(consents) != 0 && consent == nil {
			// Only allow one parent to start the process. For now just redirect to sign in, but we should have a better state here.
			www.RedirectToSignIn(w, r)
			return
		}
		hasAccess = true
		proof, err := h.dataAPI.ParentConsentProof(parentPatientID)
		if err == nil {
			if proof.SelfiePhotoID != nil {
				idProof["selfie"], err = h.mediaStore.SignedURL(*proof.SelfiePhotoID, time.Hour)
				if err != nil {
					www.InternalServerError(w, r, err)
					return
				}
			}
			if proof.GovernmentIDPhotoID != nil {
				idProof["governmentid"], err = h.mediaStore.SignedURL(*proof.GovernmentIDPhotoID, time.Hour)
				if err != nil {
					www.InternalServerError(w, r, err)
					return
				}
			}
		}
	}
	if consent == nil {
		consent = &common.ParentalConsent{}
	}

	if !hasAccess {
		if !environment.IsDev() {
			www.RedirectToSignIn(w, r)
			return
		}
		// In dev let it work anyway but log it so it's obviousl what's happening
		golog.Errorf("Token is invalid but allowing in dev")
	}

	child, err := h.dataAPI.Patient(childPatientID, true)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	// Already approved so jump straight to medical record
	if child.HasParentalConsent {
		http.Redirect(w, r, fmt.Sprintf("/pc/%d/medrecord", childPatientID), http.StatusSeeOther)
		return
	}

	www.TemplateResponse(w, http.StatusOK, h.template, &parentalConsentContext{
		Environment: environment.GetCurrent(),
		Hydration: &parentalConsentHydration{
			ChildDetails: &patientContext{
				PatientID: child.ID.String(),
				FirstName: child.FirstName,
				Gender:    child.Gender,
			},
			IsParentSignedIn: account != nil,
			ParentalConsent: &consentContext{
				Consented:    consent.Consented,
				Relationship: consent.Relationship,
			},
			IdentityVerificationImages: &identitiyImageContext{
				Types: idProof,
			},
		},
	})
}
