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
)

type patientParentHandler struct {
	dataAPI            api.DataAPI
	mediaStore         *media.Store
	expirationDuration time.Duration
}

type patientParentRequest struct {
	PatientID int64 `schema:"patient_id,required"`
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
) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&patientParentHandler{
					dataAPI:            dataAPI,
					mediaStore:         mediaStore,
					expirationDuration: expirationDuration,
				}), api.RoleDoctor, api.RoleCC),
		httputil.Get)
}

func (p *patientParentHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	var rd patientParentRequest
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = rd

	par := conc.NewParallel()

	var doctor *common.Doctor
	par.Go(func() error {
		var err error
		doctor, err = p.dataAPI.GetDoctorFromAccountID(ctxt.AccountID)
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

	ctxt.RequestCache[apiservice.Doctor] = doctor
	ctxt.RequestCache[apiservice.Patient] = patient

	if ctxt.Role == api.RoleDoctor {
		if err := apiservice.ValidateDoctorAccessToPatientFile(
			r.Method,
			ctxt.Role,
			doctor.ID.Int64(),
			patient.ID.Int64(),
			p.dataAPI,
		); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (p *patientParentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patient := ctxt.RequestCache[apiservice.Patient].(*common.Patient)

	consents, err := p.dataAPI.ParentalConsent(patient.ID.Int64())
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	pItems := make([]*parentItem, len(consents))
	for i, consent := range consents {
		parent, err := p.dataAPI.GetPatientFromID(consent.ParentPatientID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		proof, err := p.dataAPI.ParentConsentProof(parent.ID.Int64())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		pItems[i] = &parentItem{
			ID:           parent.ID.Int64(),
			FirstName:    parent.FirstName,
			LastName:     parent.LastName,
			DOB:          parent.DOB.String(),
			Gender:       strings.Title(parent.Gender),
			Relationship: consent.Relationship,
			Email:        parent.Email,
		}

		if len(parent.PhoneNumbers) > 0 {
			pItems[i].CellPhone = parent.PhoneNumbers[0].Phone.String()
		}

		if proof.SelfiePhotoID != nil {
			signedURL, err := p.mediaStore.SignedURL(*proof.SelfiePhotoID, p.expirationDuration)
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			pItems[i].Proof.SelfiePhotoURL = signedURL
		}

		if proof.GovernmentIDPhotoID != nil {
			signedURL, err := p.mediaStore.SignedURL(*proof.GovernmentIDPhotoID, p.expirationDuration)
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			pItems[i].Proof.GovernmentIDPhotoURL = signedURL
		}
	}

	httputil.JSONResponse(w, http.StatusOK, &patientParentResponse{
		Parents: pItems,
	})
}
