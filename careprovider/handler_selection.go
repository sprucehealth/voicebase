package careprovider

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/views"
)

const (
	defaultSelectionCount = 3
	selectionNamespace    = "care_provider_selection"
)

type selectionHandler struct {
	dataAPI        api.DataAPI
	selectionCount int
	apiDomain      string
}

type selectionRequest struct {
	StateCode      string `schema:"state_code"`
	Zipcode        string `schema:"zip_code"`
	PathwayTag     string `schema:"pathway_id"`
	CareProviderID int64  `schema:"care_provider_id"`
}

type selectionResponse struct {
	Message string       `json:"message,omitempty"`
	Options []views.View `json:"options"`
}

func (s *selectionRequest) Validate() error {
	if len(s.StateCode) != 2 {
		return fmt.Errorf("expected a state code to be maximum of two characters, instead got %d", len(s.StateCode))
	}
	if s.PathwayTag == "" {
		return fmt.Errorf("missing pathway tag")
	}
	return nil
}

type firstAvailableSelection struct {
	Type        string   `json:"type"`
	ImageURLs   []string `json:"image_urls"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ButtonTitle string   `json:"button_title"`
}

func (f *firstAvailableSelection) TypeName() string {
	return "first_available"
}

func (f *firstAvailableSelection) Validate(namespace string) error {
	f.Type = namespace + ":" + f.TypeName()
	if f.Title == "" {
		return errors.New("title is required")
	}
	if f.ButtonTitle == "" {
		return errors.New("button_title is required")
	}
	if len(f.ImageURLs) == 0 {
		return errors.New("image_urls are required")
	}
	return nil
}

type careProviderSelection struct {
	Type           string `json:"type"`
	ImageURL       string `json:"image_url"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	ButtonTitle    string `json:"button_title"`
	CareProviderID int64  `json:"care_provider_id,string"`
}

func (c *careProviderSelection) TypeName() string {
	return "care_provider"
}

func (c *careProviderSelection) Validate(namespace string) error {
	c.Type = namespace + ":" + c.TypeName()
	if c.Title == "" {
		return errors.New("title is required")
	}
	if c.ButtonTitle == "" {
		return errors.New("button_title is required")
	}
	if c.ImageURL == "" {
		return errors.New("image_url is required")
	}
	if c.CareProviderID == 0 {
		return errors.New("care_provider_id is required")
	}

	return nil
}

// NewSelectionHandler returns an initialized instance of selectionHandler
func NewSelectionHandler(dataAPI api.DataAPI, apiDomain string, selectionCount int) httputil.ContextHandler {
	if selectionCount == 0 {
		selectionCount = defaultSelectionCount
	}

	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&selectionHandler{
				dataAPI:        dataAPI,
				apiDomain:      apiDomain,
				selectionCount: selectionCount,
			}), httputil.Get)
}

func (c *selectionHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var rd selectionRequest
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	// TEMP HACK: Due to a client issue for the GALDERMA work we will assume a missing pathway_id is health_condition_acne
	if rd.PathwayTag == "" {
		rd.PathwayTag = api.AcnePathwayTag
	}

	if err := rd.Validate(); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	// if care provider has been specified, check if the care provider
	// is eligible to see patients for the pathway, state combination
	var msg string
	if rd.CareProviderID > 0 {
		eligible, err := c.dataAPI.CareProviderEligible(rd.CareProviderID, api.RoleDoctor, rd.StateCode, rd.PathwayTag)
		if err != nil {
			golog.Errorf(err.Error())
		}

		doctor, err := c.dataAPI.Doctor(rd.CareProviderID, true)
		if err != nil {
			golog.Errorf(err.Error())
		}

		if !eligible {
			// populate message to indicate to patient that the doctor
			// is not eligible to treat patient.
			state, _, err := c.dataAPI.State(rd.StateCode)
			if err != nil {
				golog.Errorf(err.Error())
			}

			pathway, err := c.dataAPI.PathwayForTag(rd.PathwayTag, api.PONone)
			if err != nil {
				golog.Errorf(err.Error())
			}

			if state != "" && doctor != nil && pathway != nil {
				msg = fmt.Sprintf("Sorry, %s is not current treating patients for %s in %s. Please select from another board-certified dermatologist below, or choose \"First Available\".",
					doctor.ShortDisplayName, strings.ToLower(pathway.Name), state)
			}
		} else if doctor != nil {
			response := &selectionResponse{
				Options: []views.View{
					&careProviderSelection{
						ImageURL:       app_url.ThumbnailURL(c.apiDomain, api.RoleDoctor, doctor.ID.Int64()),
						Title:          doctor.ShortDisplayName,
						Description:    doctor.LongTitle,
						ButtonTitle:    fmt.Sprintf("Choose %s", doctor.ShortDisplayName),
						CareProviderID: doctor.ID.Int64(),
					},
				},
			}

			// validate all views
			for _, selectionView := range response.Options {
				if err := selectionView.Validate(selectionNamespace); err != nil {
					apiservice.WriteError(ctx, err, w, r)
					return
				}
			}

			httputil.JSONResponse(w, http.StatusOK, response)
			return
		}
	}

	account, _ := apiservice.CtxAccount(ctx)
	doctorIDs, err := c.pickNDoctors(c.selectionCount, &rd, account)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	p := conc.NewParallel()

	// pick N doctors and the imageURLs for the first available option
	// in parallel.
	doctors := make([]*common.Doctor, 0, c.selectionCount)
	p.Go(func() error {
		var err error
		doctors, err = c.dataAPI.Doctors(doctorIDs)
		return errors.Trace(err)
	})

	var imageURLs []string
	p.Go(func() error {
		var err error
		imageURLs, err = c.randomlyPickDoctorThumbnails(6, doctorIDs)
		return errors.Trace(err)
	})

	if err := p.Wait(); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// populate views
	response := &selectionResponse{
		Message: msg,
		Options: make([]views.View, 1+len(doctors)),
	}

	response.Options[0] = &firstAvailableSelection{
		ImageURLs:   imageURLs,
		Title:       "First Available",
		Description: "Choose this option for a response within 24 hours. You'll be treated by the first available doctor on Spruce.",
		ButtonTitle: "Choose First Available",
	}
	for i, doctor := range doctors {
		response.Options[i+1] = &careProviderSelection{
			ImageURL:       app_url.ThumbnailURL(c.apiDomain, api.RoleDoctor, doctor.ID.Int64()),
			Title:          doctor.ShortDisplayName,
			Description:    doctor.LongTitle,
			ButtonTitle:    fmt.Sprintf("Choose %s", doctor.ShortDisplayName),
			CareProviderID: doctor.ID.Int64(),
		}
	}

	// validate all views
	for _, selectionView := range response.Options {
		if err := selectionView.Validate(selectionNamespace); err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}

func (c *selectionHandler) randomlyPickDoctorThumbnails(n int, pickedDoctorList []int64) ([]string, error) {
	return RandomDoctorURLs(n, c.dataAPI, c.apiDomain, pickedDoctorList)
}

func (c *selectionHandler) pickNDoctors(n int, rd *selectionRequest, account *common.Account) ([]int64, error) {
	careProvidingStateID, err := c.dataAPI.GetCareProvidingStateID(rd.StateCode, rd.PathwayTag)
	if api.IsErrNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	doctorIDs := make([]int64, 0, n)

	// if authenticated, first include
	// any eligible doctors from your past cases
	if account != nil {
		// only patient is allowed to access this API in authenticated mode
		if account.Role != api.RolePatient {
			return nil, apiservice.NewAccessForbiddenError()
		}

		patientID, err := c.dataAPI.GetPatientIDFromAccountID(account.ID)
		if err != nil {
			return nil, err
		}

		cases, err := c.dataAPI.GetCasesForPatient(patientID, common.SubmittedPatientCaseStates())
		if err != nil {
			return nil, err
		}

		caseIDs := make([]int64, len(cases))
		for i, pc := range cases {
			caseIDs[i] = pc.ID.Int64()
		}

		careTeamsByCase, err := c.dataAPI.CaseCareTeams(caseIDs)
		if err != nil {
			return nil, err
		}

		// identify all doctors across care teams
		var doctorsInCareTeams []int64
		for _, careTeam := range careTeamsByCase {
			for _, assignment := range careTeam.Assignments {
				if assignment.ProviderRole == api.RoleDoctor && assignment.Status == api.StatusActive {
					doctorsInCareTeams = append(doctorsInCareTeams, assignment.ProviderID)
				}
			}
		}

		// determine which of these doctors are eligible for this pathway and state combination
		eligibleDoctorIDs, err := c.dataAPI.EligibleDoctorIDs(doctorsInCareTeams, careProvidingStateID)
		if err != nil {
			return nil, err
		}

		// if the number of eligible doctors from the patient's care teams
		// is greater than the number of required doctors, then just pick the first
		// n doctors
		if len(eligibleDoctorIDs) >= n {
			return eligibleDoctorIDs[:n], nil
		}

		doctorIDs = append(doctorIDs, eligibleDoctorIDs...)
	}

	remainingNumToPick := n - len(doctorIDs)

	// get a list of all doctorIDs available for this pathway, state combination
	availableDoctorIDs, err := c.dataAPI.DoctorIDsInCareProvidingState(careProvidingStateID)
	if err != nil {
		return nil, err
	}

	// create a set of picked doctorIDs for quick lookup
	pickedDoctorIDSet := make(map[int64]bool)
	for _, pickedDoctorID := range doctorIDs {
		pickedDoctorIDSet[pickedDoctorID] = true
	}

	// filter out from the list of availableDoctors the ones that have already been picked
	filteredAvailableDoctorIDs := make([]int64, 0, len(availableDoctorIDs))
	for _, availableDoctorID := range availableDoctorIDs {
		if !pickedDoctorIDSet[availableDoctorID] {
			filteredAvailableDoctorIDs = append(filteredAvailableDoctorIDs, availableDoctorID)
			// mark the doctor as being picked to ensure that it doesn't
			// get picked again
			pickedDoctorIDSet[availableDoctorID] = true
		}

	}
	numAvailableDoctors := len(filteredAvailableDoctorIDs)

	switch {

	case remainingNumToPick == numAvailableDoctors:
		// optimize for the case where the remaining number of required
		// doctors equals the number of available doctors to avoid a bunch of
		// random number processing for nothing
		return append(doctorIDs, filteredAvailableDoctorIDs...), nil

	case remainingNumToPick > numAvailableDoctors:
		// if in the event the number of available doctors
		// is less than the required number, minimize expectations
		// of the required number of doctors
		remainingNumToPick = numAvailableDoctors
		fallthrough

	case remainingNumToPick < numAvailableDoctors:
		pickedDoctorIDSet = make(map[int64]bool)
		for remainingNumToPick > 0 {
			toPick := filteredAvailableDoctorIDs[rand.Intn(numAvailableDoctors)]
			if !pickedDoctorIDSet[toPick] {
				doctorIDs = append(doctorIDs, toPick)
				pickedDoctorIDSet[toPick] = true
				remainingNumToPick--
			}
		}

	}

	return doctorIDs, nil
}
