package doctor_treatment_plan

import (
	"net/http"
	"sort"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/responses"
	"golang.org/x/net/context"
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
) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.AuthorizationRequired(&doctorFavoriteTreatmentPlansHandler{
				dataAPI:    dataAPI,
				erxAPI:     erxAPI,
				mediaStore: mediaStore,
			})),
		httputil.Get, httputil.Post, httputil.Delete, httputil.Put)
}

type DoctorFavoriteTreatmentPlansRequestData struct {
	FavoriteTreatmentPlanID int64                            `json:"favorite_treatment_plan_id" schema:"favorite_treatment_plan_id"`
	FavoriteTreatmentPlan   *responses.FavoriteTreatmentPlan `json:"favorite_treatment_plan"`
	TreatmentPlanID         int64                            `json:"treatment_plan_id,omitempty,string"`
	PathwayTag              string                           `json:"pathway_id" schema:"pathway_id"`
}

type DoctorFavoriteTreatmentPlansResponseData struct {
	FavoriteTreatmentPlansByPathway []*responses.PathwayFTPGroup       `json:"favorite_treatment_plans_by_pathway,omitempty"`
	FavoriteTreatmentPlans          []*responses.FavoriteTreatmentPlan `json:"favorite_treatment_plans,omitempty"`
	FavoriteTreatmentPlan           *responses.FavoriteTreatmentPlan   `json:"favorite_treatment_plan,omitempty"`
}

func (d *doctorFavoriteTreatmentPlansHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	doctor, err := d.dataAPI.GetDoctorFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctor] = doctor

	requestData := &DoctorFavoriteTreatmentPlansRequestData{}
	if err := apiservice.DecodeRequestData(requestData, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	requestCache[apiservice.CKRequestData] = requestData

	if requestData.FavoriteTreatmentPlanID > 0 {
		// ensure that the doctor is a member of the favorite treatment plan
		favoriteTreatmentPlan, err := d.dataAPI.FavoriteTreatmentPlan(requestData.FavoriteTreatmentPlanID)
		if err != nil {
			return false, err
		}
		requestCache[apiservice.CKFavoriteTreatmentPlan] = favoriteTreatmentPlan
	}

	if requestData.TreatmentPlanID > 0 {
		// ensure that the doctor has access to the patient file
		treatmentPlan, err := d.dataAPI.GetTreatmentPlan(requestData.TreatmentPlanID, doctor.ID.Int64())
		if err != nil {
			return false, err
		}
		requestCache[apiservice.CKTreatmentPlan] = treatmentPlan

		// ensure that the doctor owns the treatment plan
		if treatmentPlan.DoctorID.Int64() != doctor.ID.Int64() {
			return false, apiservice.NewAccessForbiddenError()
		}

		if err := apiservice.ValidateAccessToPatientCase(r.Method, account.Role, doctor.ID.Int64(), treatmentPlan.PatientID, treatmentPlan.PatientCaseID.Int64(), d.dataAPI); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (d *doctorFavoriteTreatmentPlansHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	doctor := requestCache[apiservice.CKDoctor].(*common.Doctor)
	requestData := requestCache[apiservice.CKRequestData].(*DoctorFavoriteTreatmentPlansRequestData)

	switch r.Method {
	case httputil.Get:
		d.getFavoriteTreatmentPlans(ctx, w, r, doctor, requestData)
	case httputil.Post, httputil.Put:
		d.addOrUpdateFavoriteTreatmentPlan(ctx, w, r, doctor, requestData)
	case httputil.Delete:
		d.deleteFavoriteTreatmentPlan(ctx, w, r, doctor, requestData)
	default:
		http.NotFound(w, r)
	}
}

func (d *doctorFavoriteTreatmentPlansHandler) getFavoriteTreatmentPlans(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	doctor *common.Doctor,
	requestData *DoctorFavoriteTreatmentPlansRequestData,
) {
	if requestData.PathwayTag != "" {
		_, err := d.dataAPI.PathwayForTag(requestData.PathwayTag, api.PONone)
		if api.IsErrNotFound(err) {
			apiservice.WriteBadRequestError(ctx, err, w, r)
		}
	}

	// no favorite treatment plan id specified in which case return all for the requested pathway
	if requestData.FavoriteTreatmentPlanID == 0 {
		pathwayFTPGroups, err := d.pathwayFTPGroupsForDoctor(doctor.ID.Int64(), requestData.PathwayTag)
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		// TODO: Remove this once the doctors have upgraded to the new app
		if requestData.PathwayTag != "" {
			httputil.JSONResponse(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: pathwayFTPGroups[0].FTPs})
			return
		}
		httputil.JSONResponse(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlansByPathway: pathwayFTPGroups})
		return
	}

	requestCache := apiservice.MustCtxCache(ctx)
	ftp := requestCache[apiservice.CKFavoriteTreatmentPlan].(*common.FavoriteTreatmentPlan)
	ftpRes, err := responses.TransformFTPToResponse(d.dataAPI, d.mediaStore, scheduledMessageMediaExpirationDuration, ftp, requestData.PathwayTag)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: ftpRes})
}

func (d *doctorFavoriteTreatmentPlansHandler) addOrUpdateFavoriteTreatmentPlan(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	doctor *common.Doctor,
	req *DoctorFavoriteTreatmentPlansRequestData,
) {
	account := apiservice.MustCtxAccount(ctx)

	ftp, err := responses.TransformFTPFromResponse(d.dataAPI, req.FavoriteTreatmentPlan, doctor.ID.Int64(), account.Role)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// validate treatments being added
	if ftp.TreatmentList != nil {
		if err := validateTreatments(ftp.TreatmentList.Treatments, d.dataAPI, d.erxAPI, doctor.DoseSpotClinicianID); err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	// this means that the favorite treatment plan was created
	// in the context of a treatment plan so compare the two
	// to ensure they are equal
	if req.TreatmentPlanID != 0 {
		requestCache := apiservice.MustCtxCache(ctx)
		tp := requestCache[apiservice.CKTreatmentPlan].(*common.TreatmentPlan)

		// if the pathway_tag is not specified, pick it up from the treatment_plan_id
		// that is linked to the patient case
		if req.PathwayTag == "" {
			patientCase, err := d.dataAPI.GetPatientCaseFromID(tp.PatientCaseID.Int64())
			if err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}

			req.PathwayTag = patientCase.PathwayTag
		}

		if !ftp.EqualsTreatmentPlan(tp) {
			apiservice.WriteValidationError(ctx, "Cannot associate a favorite treatment plan with a treatment plan when the contents of the two don't match", w, r)
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
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	// ensure that any tokens in the note are valid
	t := newTokenizerForValidation('{', '}')
	if err := t.validate(ftp.Note); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// prepare the favorite treatment plan to have a creator id
	id := doctor.ID.Int64()
	ftp.CreatorID = &id

	if _, err := d.dataAPI.InsertFavoriteTreatmentPlan(ftp, req.PathwayTag, req.TreatmentPlanID); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if err := d.dataAPI.SetFavoriteTreatmentPlanScheduledMessages(ftp.ID.Int64(), ftp.ScheduledMessages); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	ftpRes, err := responses.TransformFTPToResponse(d.dataAPI, d.mediaStore, scheduledMessageMediaExpirationDuration, ftp, req.PathwayTag)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlan: ftpRes})
}

func (d *doctorFavoriteTreatmentPlansHandler) deleteFavoriteTreatmentPlan(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	doctor *common.Doctor,
	req *DoctorFavoriteTreatmentPlansRequestData,
) {
	if req.FavoriteTreatmentPlanID == 0 {
		apiservice.WriteValidationError(ctx, "favorite_treatment_plan_id must be specifeid", w, r)
		return
	}

	// TODO: for now default to acne if no pathway specified
	if req.PathwayTag == "" {
		req.PathwayTag = api.AcnePathwayTag
	}

	if err := d.dataAPI.DeleteFavoriteTreatmentPlanScheduledMessages(req.FavoriteTreatmentPlanID); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	if err := d.dataAPI.DeleteFavoriteTreatmentPlan(req.FavoriteTreatmentPlanID, doctor.ID.Int64(), req.PathwayTag); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// echo back updated list of favorite treatment plans
	pathwayFTPGroups, err := d.pathwayFTPGroupsForDoctor(doctor.ID.Int64(), req.PathwayTag)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// TODO: Remove this once the doctors have upgraded to the new app
	if req.PathwayTag != "" {
		httputil.JSONResponse(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlans: pathwayFTPGroups[0].FTPs})
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &DoctorFavoriteTreatmentPlansResponseData{FavoriteTreatmentPlansByPathway: pathwayFTPGroups})
}

func (d *doctorFavoriteTreatmentPlansHandler) pathwayFTPGroupsForDoctor(id int64, pathwayTag string) ([]*responses.PathwayFTPGroup, error) {
	ftpsByPathway, err := d.dataAPI.FavoriteTreatmentPlansForDoctor(id, pathwayTag)
	if err != nil {
		return nil, err
	}
	pathwayFTPGroups := make([]*responses.PathwayFTPGroup, len(ftpsByPathway))
	var groupIndex int
	for pathwayTag, ftps := range ftpsByPathway {
		ftpsResponses := make([]*responses.FavoriteTreatmentPlan, len(ftps))
		for i, ftp := range ftps {
			ftpsResponses[i], err = responses.TransformFTPToResponse(d.dataAPI, d.mediaStore, scheduledMessageMediaExpirationDuration, ftp, pathwayTag)
			if err != nil {
				return nil, err
			}
		}
		pathway, err := d.dataAPI.PathwayForTag(pathwayTag, api.PONone)
		if err != nil {
			return nil, err
		}
		pathwayFTPGroups[groupIndex] = &responses.PathwayFTPGroup{
			FTPs:        ftpsResponses,
			PathwayName: pathway.Name,
			PathwayTag:  pathway.Tag,
		}
		groupIndex++
	}

	// If the user asked for a specific pathway and we didn't find any FTPS make sure we at least return an empty set
	if pathwayTag != "" && len(pathwayFTPGroups) == 0 {
		pathway, err := d.dataAPI.PathwayForTag(pathwayTag, api.PONone)
		if err != nil {
			return nil, err
		}
		pathwayFTPGroups = []*responses.PathwayFTPGroup{
			&responses.PathwayFTPGroup{
				PathwayName: pathway.Name,
				PathwayTag:  pathway.Tag,
			},
		}
	}
	sort.Sort(responses.PathwayFTPGroupByPathwayName(pathwayFTPGroups))
	return pathwayFTPGroups, nil
}
