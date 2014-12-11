package diagnosis

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/libs/httputil"
)

var (
	acneDiagnosisCodes = []string{"L70.0", "L71.9", "L71.0"}
)

type searchHandler struct {
	dataAPI api.DataAPI
}

type DiagnosisSearchResult struct {
	Sections []*ResultSection `json:"result_sections"`
}

type ResultSection struct {
	Title string        `json:"title"`
	Items []*ResultItem `json:"items"`
}

type ResultItem struct {
	Title     string             `json:"title"`
	Subtitle  string             `json:"subtitle,omitempty"`
	Diagnosis *AbridgedDiagnosis `json:"abridged_diagnosis"`
}

type AbridgedDiagnosis struct {
	CodeID     int64  `json:"code_id,string"`
	Code       string `json:"display_diagnosis_code"`
	Title      string `json:"title"`
	Synonyms   string `json:"synonyms,omitempty"`
	HasDetails bool   `json:"has_details"`
}

func NewSearchHandler(dataAPI api.DataAPI) http.Handler {
	return apiservice.SupportedRoles(
		httputil.SupportedMethods(
			apiservice.NoAuthorizationRequired(&searchHandler{
				dataAPI: dataAPI,
			}), []string{"GET"}), []string{api.DOCTOR_ROLE})
}

func (s *searchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// FIX ME : for now do nothing just short circuit to return the same result until
	// we build search
	diagnosisMap, err := s.dataAPI.DiagnosisForCodes(acneDiagnosisCodes)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	result := &DiagnosisSearchResult{
		Sections: []*ResultSection{
			&ResultSection{
				Title: "Common Acne Diagnoses",
			},
		},
	}

	items := make([]*ResultItem, len(diagnosisMap))
	for i, code := range acneDiagnosisCodes {
		diagnosis := diagnosisMap[code]
		items[i] = &ResultItem{
			Title: diagnosis.Description,
			Diagnosis: &AbridgedDiagnosis{
				CodeID:     diagnosis.ID,
				Code:       diagnosis.Code,
				Title:      diagnosis.Description,
				HasDetails: true,
			},
		}
	}
	result.Sections[0].Items = items

	apiservice.WriteJSON(w, result)
}
