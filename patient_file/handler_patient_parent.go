package patient_file

import (
	"net/http"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
	"golang.org/x/net/context"
)

type patientParentHandler struct {
	dataAPI            api.DataAPI
	mediaStore         *media.Store
	expirationDuration time.Duration
}

type patientParentRequest struct {
	PatientID common.PatientID `schema:"patient_id,required"`
}

type consentProof struct {
	SelfiePhotoURL       string `json:"selfie_photo_url"`
	GovernmentIDPhotoURL string `json:"governmentid_photo_url"`
}

type parentItem struct {
	ID           int64        `json:"id,string"`
	FirstName    string       `json:"first_name"`
	LastName     string       `json:"last_name"`
	DOB          string       `json:"dob"`
	Gender       string       `json:"gender"`
	Relationship string       `json:"relationship"`
	Email        string       `json:"email"`
	CellPhone    string       `json:"cell_phone"`
	Proof        consentProof `json:"consent_proof"`
}

type patientParentResponse struct {
	Parents []*parentItem `json:"parents"`
}

// NewPatientParentHandler returns a handler that allows a doctor or
// care coordinator to get parent information for a given patient if
// the information exists in our system. Note that only a doctor
// that is part of the patient's care team can get the information.
func NewPatientParentHandler(
	dataAPI api.DataAPI,
	mediaStore *media.Store,
	expirationDuration time.Duration,
) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(
					&patientParentHandler{
						dataAPI:            dataAPI,
						mediaStore:         mediaStore,
						expirationDuration: expirationDuration,
					})),
			api.RoleDoctor, api.RoleCC),
		httputil.Get)
}

func (p *patientParentHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	var rd patientParentRequest
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	requestCache[apiservice.CKRequestData] = rd

	par := conc.NewParallel()

	var doctor *common.Doctor
	par.Go(func() error {
		var err error
		doctor, err = p.dataAPI.GetDoctorFromAccountID(account.ID)
		return errors.Trace(err)
	})

	var patient *common.Patient
	par.Go(func() error {
		var err error
		patient, err = p.dataAPI.Patient(rd.PatientID, true)
		return errors.Trace(err)
	})

	if err := par.Wait(); err != nil {
		return false, errors.Trace(err)
	}

	requestCache[apiservice.CKDoctor] = doctor
	requestCache[apiservice.CKPatient] = patient

	if account.Role == api.RoleDoctor {
		if err := apiservice.ValidateDoctorAccessToPatientFile(
			r.Method,
			account.Role,
			doctor.ID.Int64(),
			patient.ID,
			p.dataAPI,
		); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (p *patientParentHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	patient := requestCache[apiservice.CKPatient].(*common.Patient)

	consents, err := p.dataAPI.ParentalConsent(patient.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	pItems := make([]*parentItem, 0, len(consents))
	for _, consent := range consents {
		parent, err := p.dataAPI.GetPatientFromID(consent.ParentPatientID)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		// only include the parent information for the ones that
		// provided consent.
		proof, err := p.dataAPI.ParentConsentProof(parent.ID)
		if api.IsErrNotFound(err) {
			continue
		} else if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		pItem := &parentItem{
			ID:           parent.ID.Int64(),
			FirstName:    parent.FirstName,
			LastName:     parent.LastName,
			DOB:          parent.DOB.String(),
			Gender:       strings.Title(parent.Gender),
			Relationship: consent.Relationship,
			Email:        parent.Email,
		}

		pItems = append(pItems, pItem)

		if len(parent.PhoneNumbers) > 0 {
			pItem.CellPhone = parent.PhoneNumbers[0].Phone.String()
		}

		if proof.SelfiePhotoID != nil {
			signedURL, err := p.mediaStore.SignedURL(*proof.SelfiePhotoID, p.expirationDuration)
			if err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}

			pItem.Proof.SelfiePhotoURL = signedURL
		}

		if proof.GovernmentIDPhotoID != nil {
			signedURL, err := p.mediaStore.SignedURL(*proof.GovernmentIDPhotoID, p.expirationDuration)
			if err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}

			pItem.Proof.GovernmentIDPhotoURL = signedURL
		}
	}

	httputil.JSONResponse(w, http.StatusOK, &patientParentResponse{
		Parents: pItems,
	})
}
