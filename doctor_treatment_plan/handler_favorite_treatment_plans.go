package doctor_treatment_plan

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/responses"
)

type doctorFavoriteTreatmentPlansHandler struct {
	dataAPI    api.DataAPI
	erxAPI     erx.ERxAPI
	mediaStore *media.Store
}

func NewDoctorFavoriteTreatmentPlansHandler(
	dataAPI api.DataAPI,
	erxAPI erx.ERxAPI,
	mediaStore *media.Store,
) http.Handler {
	return httputil.SupportedMethods(
		apiservice.AuthorizationRequired(&doctorFavoriteTreatmentPlansHandler{
			dataAPI:    dataAPI,
			erxAPI:     erxAPI,
			mediaStore: mediaStore,
		}), []string{"GET", "POST", "DELETE", "PUT"})
}

type DoctorFavoriteTreatmentPlansRequestData struct {
	FavoriteTreatmentPlanID int64                            `json:"favorite_treatment_plan_id" schema:"favorite_treatment_plan_id"`
	FavoriteTreatmentPlan   *responses.FavoriteTreatmentPlan `json:"favorite_treatment_plan"`
	TreatmentPlanID         int64                            `json:"treatment_plan_id,omitempty,string"`
	PathwayTag              string                           `json:"pathway_id" schema:"pathway_id"`
}

type DoctorFavoriteTreatmentPlansResponseData struct {
	FavoriteTreatmentPlans []*responses.FavoriteTreatmentPlan `json:"favorite_treatment_plans,omitempty"`
	FavoriteTreatmentPlan  *responses.FavoriteTreatmentPlan   `json:"favorite_treatment_plan,omitempty"`
}

func (d *doctorFavoriteTreatmentPlansHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	doctor, err := d.dataAPI.GetDoctorFromAccountID(apiservice.GetContext(r).AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.Doctor] = doctor

	requestData := &DoctorFavoriteTreatmentPlansRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	ctxt.RequestCache[apiservice.RequestData] = requestData

	if requestData.FavoriteTreatmentPlanID > 0 {
		// ensure that the doctor is a member of the favorite treatment plan
		favoriteTreatmentPlan, err := d.dataAPI.FavoriteTreatmentPlan(requestData.FavoriteTreatmentPlanID)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.FavoriteTreatmentPlan] = favoriteTreatmentPlan
	}

	if requestData.TreatmentPlanID > 0 {
		// ensure that the doctor has access to the patient file
		treatmentPlan, err := d.dataAPI.GetTreatmentPlan(requestData.TreatmentPlanID, doctor.DoctorID.Int64())
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.TreatmentPlan] = treatmentPlan

		// ensure that the doctor owns the treatment plan
		if treatmentPlan.DoctorID.Int64() != doctor.DoctorID.Int64() {
			return false, apiservice.NewAccessForbiddenError()
		}

		if err := apiservice.ValidateAccessToPatientCase(r.Method, ctxt.Role, doctor.DoctorID.Int64(), treatmentPlan.PatientID, treatmentPlan.PatientCaseID.Int64(), d.dataAPI); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (d *doctorFavoriteTreatmentPlansHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	doctor := ctxt.RequestCache[apiservice.Doctor].(*common.Doctor)
	requestData := ctxt.RequestCache[apiservice.RequestData].(*DoctorFavoriteTreatmentPlansRequestData)

	switch r.Method {
	case httputil.Get:
		d.getFavoriteTreatmentPlans(w, r, doctor, requestData)
	case httputil.Post, httputil.Put:
		d.addOrUpdateFavoriteTreatmentPlan(w, r, doctor, requestData)
	case httputil.Delete:
		d.deleteFavoriteTreatmentPlan(w, r, doctor, requestData)
	default:
		http.NotFound(w, r)
	}
}

func (d *doctorFavoriteTreatmentPlansHandler) getFavoriteTreatmentPlans(
	w http.ResponseWriter,
	r *http.Request,
	doctor *common.Doctor,
	requestData *DoctorFavoriteTreatmentPlansRequestData,
) {
	// no favorite treatment plan id specified in which case return all for the requested pathway
	if requestData.FavoriteTreatmentPlanID == 0 {
		// TODO: for now default to acne if no pathway specified
		if requestData.PathwayTag == "" {
			requestData.PathwayTag = api.AcnePathwayTag
		}

		ftps, err := d.dataAPI.FavoriteTreatmentPlansForDoctor(doctor.DoctorID.Int64(), requestData.PathwayTag)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		ftpsRes := make([]*responses.FavoriteTreatmentPlan, len(ftps))
		for i, ftp := range ftps {
			ftpsRes[i], err = responses.TransformFTPToResponse(d.dataAPI, d.mediaStore, scheduledMessageMediaExpirationDuration, ftp, requestData.PathwayTag)
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		}
		httputil.JSONResponse(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: ftpsRes})
		return
	}

	ftp := apiservice.GetContext(r).RequestCache[apiservice.FavoriteTreatmentPlan].(*common.FavoriteTreatmentPlan)
	ftpRes, err := responses.TransformFTPToResponse(d.dataAPI, d.mediaStore, scheduledMessageMediaExpirationDuration, ftp, requestData.PathwayTag)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: ftpRes})
}

func (d *doctorFavoriteTreatmentPlansHandler) addOrUpdateFavoriteTreatmentPlan(
	w http.ResponseWriter,
	r *http.Request,
	doctor *common.Doctor,
	req *DoctorFavoriteTreatmentPlansRequestData) {
	ctx := apiservice.GetContext(r)
	ftp, err := responses.TransformFTPFromResponse(d.dataAPI, req.FavoriteTreatmentPlan, doctor.DoctorID.Int64(), ctx.Role)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// validate treatments being added
	if ftp.TreatmentList != nil {
		if err := validateTreatments(
			ftp.TreatmentList.Treatments,
			d.dataAPI,
			d.erxAPI,
			doctor.DoseSpotClinicianID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	// this means that the favorite treatment plan was created
	// in the context of a treatment plan so compare the two
	// to ensure they are equal
	if req.TreatmentPlanID != 0 {
		tp := apiservice.GetContext(r).RequestCache[apiservice.TreatmentPlan].(*common.TreatmentPlan)

		// if the pathway_tag is not specified, pick it up from the treatment_plan_id
		// that is linked to the patient case
		if req.PathwayTag == "" {
			patientCase, err := d.dataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}

			req.PathwayTag = patientCase.PathwayTag
		}

		if ftp.Note == "" {
			// NOTE: Empty out the tp note before comparing the FTP and TP if the FTP note is empty.
			// Reason for this is that older clients don't send the note as part of the FTP and so the verification
			// for the contents between FTP and TP being equal will fail.
			// TODO: Remove this check once Buzz Lightyear doctor app version is deployed.
			tp.Note = ""
		}

		if !ftp.EqualsTreatmentPlan(tp) {
			apiservice.WriteValidationError("Cannot associate a favorite treatment plan with a treatment plan when the contents of the two don't match", w, r)
			return
		}
	} else {
		// TODO: Don't assume acne
		if req.PathwayTag == "" {
			req.PathwayTag = api.AcnePathwayTag
		}
	}

	// ensure that favorite treatment plan has a name
	if err := ftp.Validate(); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	// prepare the favorite treatment plan to have a creator id
	ftp.CreatorID = doctor.DoctorID.Int64()

	if err := d.dataAPI.InsertFavoriteTreatmentPlan(ftp, req.PathwayTag, req.TreatmentPlanID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if err := d.dataAPI.SetFavoriteTreatmentPlanScheduledMessages(ftp.ID.Int64(), ftp.ScheduledMessages); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	ftpRes, err := responses.TransformFTPToResponse(d.dataAPI, d.mediaStore, scheduledMessageMediaExpirationDuration, ftp, req.PathwayTag)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: ftpRes})
}

func (d *doctorFavoriteTreatmentPlansHandler) deleteFavoriteTreatmentPlan(
	w http.ResponseWriter,
	r *http.Request,
	doctor *common.Doctor,
	req *DoctorFavoriteTreatmentPlansRequestData,
) {
	if req.FavoriteTreatmentPlanID == 0 {
		apiservice.WriteValidationError("favorite_treatment_plan_id must be specifeid", w, r)
		return
	}

	// TODO: for now default to acne if no pathway specified
	if req.PathwayTag == "" {
		req.PathwayTag = api.AcnePathwayTag
	}

	if err := d.dataAPI.DeleteFavoriteTreatmentPlanScheduledMessages(req.FavoriteTreatmentPlanID); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	if err := d.dataAPI.DeleteFavoriteTreatmentPlan(req.FavoriteTreatmentPlanID, doctor.DoctorID.Int64(), req.PathwayTag); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// echo back updated list of favorite treatment plans
	ftps, err := d.dataAPI.FavoriteTreatmentPlansForDoctor(doctor.DoctorID.Int64(), req.PathwayTag)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	ftpsRes := make([]*responses.FavoriteTreatmentPlan, len(ftps))
	for i, ftp := range ftps {
		ftpsRes[i], err = responses.TransformFTPToResponse(d.dataAPI, d.mediaStore, scheduledMessageMediaExpirationDuration, ftp, req.PathwayTag)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}
	httputil.JSONResponse(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: ftpsRes})
}
