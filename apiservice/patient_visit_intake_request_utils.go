package apiservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
)

type doctorInfo struct {
	ShortDisplayName  string
	SmallThumbnailURL string
	Description       string
}
type VisitLayoutContext struct {
	Doctor                     *doctorInfo
	Patient                    *common.Patient
	CaseName                   string
	CheckoutHeaderText         string
	SubmissionConfirmationText string
}

type templated struct {
	Templated bool `json:"is_templated"`
}

func GetPatientLayoutForPatientVisit(
	visit *common.PatientVisit,
	languageID int64,
	dataAPI api.DataAPI,
	apiDomain string,
) (*info_intake.InfoIntakeLayout, error) {
	layoutVersion, err := dataAPI.GetPatientLayout(visit.LayoutVersionID.Int64(), languageID)
	if err != nil {
		return nil, err
	}

	// first lets check if the json is templated
	var isTemplated templated
	if err := json.Unmarshal(layoutVersion.Layout, &isTemplated); err != nil {
		return nil, err
	} else if isTemplated.Templated {
		var doctor *common.Doctor

		// if it is then populate the context
		doctorMember, err := dataAPI.GetActiveCareTeamMemberForCase(api.RoleDoctor, visit.PatientCaseID.Int64())
		if err == nil {
			doctor, err = dataAPI.Doctor(doctorMember.ProviderID, true)
			if err != nil {
				return nil, err
			}
		} else if !api.IsErrNotFound(err) {
			return nil, err
		}

		patient, err := dataAPI.Patient(visit.PatientID, true)
		if err != nil {
			return nil, err
		}
		patientCase, err := dataAPI.GetPatientCaseFromID(visit.PatientCaseID.Int64())
		if err != nil {
			return nil, err
		}

		context := &VisitLayoutContext{
			Patient:  patient,
			CaseName: patientCase.Name,
		}

		// if no doctor is found then we assume that the visit
		// will be treated by the first available doctor
		if doctor == nil {
			context.Doctor = &doctorInfo{
				Description:       "First Available Doctor",
				ShortDisplayName:  "your doctor",
				SmallThumbnailURL: "",
			}
			context.CheckoutHeaderText = "In 24 hours your doctor will review your visit and create your treatment plan."
			context.SubmissionConfirmationText = "Your doctor will review your visit and respond in 24 hours."
		} else {
			context.Doctor = &doctorInfo{
				Description:       doctor.ShortDisplayName,
				ShortDisplayName:  doctor.ShortDisplayName,
				SmallThumbnailURL: app_url.ThumbnailURL(apiDomain, api.RoleDoctor, doctor.ID.Int64()),
			}
			context.CheckoutHeaderText = fmt.Sprintf("%s will review your visit and create your treatment plan.", doctor.ShortDisplayName)
			context.SubmissionConfirmationText = fmt.Sprintf("We've sent your visit to %s for review.", doctor.ShortDisplayName)
		}

		layout, err := applyLayoutToContext(context, layoutVersion.Layout)
		if err != nil {
			return nil, err
		} else if layout != nil {
			layoutVersion.Layout = layout
		}
	}

	patientVisitLayout := &info_intake.InfoIntakeLayout{}
	if err := json.Unmarshal(layoutVersion.Layout, patientVisitLayout); err != nil {
		return nil, err
	}
	return patientVisitLayout, nil
}

func applyLayoutToContext(context *VisitLayoutContext, layout []byte) ([]byte, error) {
	if context == nil {
		return nil, nil
	}

	funcMap := template.FuncMap{
		"titleDoctor": func(str string) string {
			if str == "your doctor" {
				return "Your doctor"
			}
			return str
		},
		"toLower": strings.ToLower,
	}

	var b bytes.Buffer
	tmpl, err := template.New("Layout").Funcs(funcMap).Parse(string(layout))
	if err != nil {
		return nil, err
	}
	if err := tmpl.Execute(&b, context); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
