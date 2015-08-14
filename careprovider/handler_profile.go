package careprovider

import (
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type careProviderProfileHandler struct {
	dataAPI   api.DataAPI
	apiDomain string
}

func NewProfileHandler(dataAPI api.DataAPI, apiDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			&careProviderProfileHandler{
				dataAPI:   dataAPI,
				apiDomain: apiDomain,
			}), httputil.Get)
}

func (h *careProviderProfileHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// We only have doctors for providers so provider_id is actually the doctor ID, but
	// to future compatibility have the param be provider_id.
	doctorID, err := strconv.ParseInt(r.FormValue("provider_id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	doctor, err := h.dataAPI.GetDoctorFromID(doctorID)
	if api.IsErrNotFound(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	profile, err := h.dataAPI.CareProviderProfile(doctor.AccountID.Int64())
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// If the profile full name is empty (should never be after a doctor
	// is onboarded) use the long display name. This will make it a little
	// nicer in demo, staging, and dev where the profile may not be filled out.
	if profile.FullName == "" {
		profile.FullName = doctor.LongDisplayName
	}

	role := api.RoleDoctor
	if doctor.IsCC {
		role = api.RoleCC
	}

	views := []profileView{
		&profileHeaderView{
			PhotoURL: app_url.HeroImageURL(h.apiDomain, role, doctor.ID.Int64()),
			Title:    profile.FullName,
			Subtitle: doctor.LongTitle,
		},
		&profileLargeDivider{},
	}

	if profile.WhySpruce != "" {
		views = append(views, &profileSectionView{
			Title:   "Why Spruce?",
			IconURL: app_url.IconProfileSpruceLogo,
			Views: []profileView{
				&profileTextView{
					Text: profile.WhySpruce,
				},
			},
		})
	}

	if profile.Qualifications != "" {
		views = append(views,
			&profileLargeDivider{},
			&profileSectionView{
				Title:   "Qualifications",
				IconURL: app_url.IconProfileQualifications,
				Views: []profileView{
					&profileTextView{
						Text: profile.Qualifications,
					},
				},
			},
		)
	}

	sec := &profileSectionView{
		Title:   "Education",
		IconURL: app_url.IconProfileEducation,
	}
	if profile.MedicalSchool != "" {
		sec.Views = append(sec.Views, &profileCategoryAndTitleView{
			Category: "Medical School",
			Title:    profile.MedicalSchool,
		})
	}
	if profile.Residency != "" {
		sec.Views = append(sec.Views, &profileCategoryAndTitleView{
			Category: "Residency",
			Title:    profile.Residency,
		})
	}
	if profile.Fellowship != "" {
		sec.Views = append(sec.Views, &profileCategoryAndTitleView{
			Category: "Fellowship",
			Title:    profile.Fellowship,
		})
	}
	if profile.GraduateSchool != "" {
		sec.Views = append(sec.Views, &profileCategoryAndTitleView{
			Category: "Graduate",
			Title:    profile.GraduateSchool,
		})
	}
	if profile.UndergraduateSchool != "" {
		sec.Views = append(sec.Views, &profileCategoryAndTitleView{
			Category: "Undergraduate",
			Title:    profile.UndergraduateSchool,
		})
	}
	if len(sec.Views) != 0 {
		views = append(views, &profileLargeDivider{}, sec)
	}

	if profile.Experience != "" {
		views = append(views,
			&profileLargeDivider{},
			&profileSectionView{
				Title:   "Experience",
				IconURL: app_url.IconProfileExperience,
				Views: []profileView{
					&profileTextView{
						Text: profile.Experience,
					},
				},
			},
		)
	}

	for _, v := range views {
		if err := v.Validate(); err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	httputil.JSONResponse(w, http.StatusOK, map[string][]profileView{"views": views})
}

const profileViewNamespace = "provider_profile"

type profileView interface {
	Validate() error
	TypeName() string
}

type profileHeaderView struct {
	Type     string `json:"type"`
	PhotoURL string `json:"profile_photo_url"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

func (v *profileHeaderView) TypeName() string {
	return profileViewNamespace + ":header"
}

func (v *profileHeaderView) Validate() error {
	v.Type = v.TypeName()
	return nil
}

type profileSectionView struct {
	Type    string               `json:"type"`
	Title   string               `json:"title"`
	IconURL *app_url.SpruceAsset `json:"icon_url"`
	Views   []profileView        `json:"views"`
}

func (v *profileSectionView) TypeName() string {
	return profileViewNamespace + ":section"
}

func (v *profileSectionView) Validate() error {
	v.Type = v.TypeName()
	for _, v := range v.Views {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type profileTextView struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (v *profileTextView) TypeName() string {
	return profileViewNamespace + ":text"
}

func (v *profileTextView) Validate() error {
	v.Type = v.TypeName()
	return nil
}

type profileLargeDivider struct {
	Type string `json:"type"`
}

func (v *profileLargeDivider) TypeName() string {
	return profileViewNamespace + ":large_divider"
}

func (v *profileLargeDivider) Validate() error {
	v.Type = v.TypeName()
	return nil
}

type profileCategoryAndTitleView struct {
	Type     string `json:"type"`
	Category string `json:"category"`
	Title    string `json:"title"`
}

func (v *profileCategoryAndTitleView) TypeName() string {
	return profileViewNamespace + ":category_and_title"
}

func (v *profileCategoryAndTitleView) Validate() error {
	v.Type = v.TypeName()
	return nil
}
