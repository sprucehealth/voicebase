package patient_visit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
)

type pathwayDetailsHandlerDataAPI struct {
	api.DataAPI
	pathways       map[string]*common.Pathway
	pathwayCases   []*common.PatientCase
	pathwayDoctors map[string][]*common.Doctor
	careTeams      map[int64]*common.PatientCareTeam
	itemCost       *common.ItemCost
}

// pathwayDetailsRes is a simplified response version of the pathway details handler response.
// It's not possible to use the existing response struct because it uses interfaces.
type pathwayDetailsRes struct {
	Pathways []struct {
		PathwayTag string `json:"pathway_id"`
		Screen     struct {
			Type           string        `json:"type"`
			Title          string        `json:"title"`
			ContentText    string        `json:"content_text,omitempty"`
			ContentSubtext string        `json:"content_subtext,omitempty"`
			Views          []interface{} `json:"views"`
		} `json:"screen"`
		FAQ *struct {
			Views []interface{} `json:"views"`
		} `json:"faq"`
	} `json:"pathway_details_screens"`
}

func (api *pathwayDetailsHandlerDataAPI) GetPatientIDFromAccountID(accountID int64) (int64, error) {
	return 1, nil
}

func (api *pathwayDetailsHandlerDataAPI) GetCasesForPatient(patientID int64, states []string) ([]*common.PatientCase, error) {
	return api.pathwayCases, nil
}

func (api *pathwayDetailsHandlerDataAPI) PathwaysForTags(tags []string, opts api.PathwayOption) (map[string]*common.Pathway, error) {
	ps := make(map[string]*common.Pathway, len(tags))
	for _, tag := range tags {
		if p := api.pathways[tag]; p != nil {
			ps[tag] = p
		}
	}
	return ps, nil
}

func (api *pathwayDetailsHandlerDataAPI) CaseCareTeams(caseIDs []int64) (map[int64]*common.PatientCareTeam, error) {
	return api.careTeams, nil
}

func (api *pathwayDetailsHandlerDataAPI) DoctorsForPathway(pathwayTag string, limit int) ([]*common.Doctor, error) {
	return api.pathwayDoctors[pathwayTag], nil
}

func (api *pathwayDetailsHandlerDataAPI) GetActiveItemCost(skuType string) (*common.ItemCost, error) {
	return api.itemCost, nil
}
func (api *pathwayDetailsHandlerDataAPI) SKUForPathway(pathwayTag string, category common.SKUCategoryType) (*common.SKU, error) {
	return &common.SKU{}, nil
}
func (api *pathwayDetailsHandlerDataAPI) AvailableDoctorIDs(n int) ([]int64, error) {
	return []int64{1, 2, 3}, nil
}

func TestPathwayDetailsHandler(t *testing.T) {
	dataAPI := &pathwayDetailsHandlerDataAPI{
		pathways: map[string]*common.Pathway{
			"acne": {
				ID:   1,
				Tag:  "acne",
				Name: "Acne",
				Details: &common.PathwayDetails{
					WhatIsIncluded: []string{"Cheese"},
					WhoWillTreatMe: "George Carlin",
					RightForMe:     "Probably",
					DidYouKnow:     []string{"BEEEEES"},
					FAQ: []common.FAQ{
						{Question: "Why?", Answer: "Because"},
					},
				},
			},
			"arachnophobia": {
				ID:   2,
				Tag:  "arachnophobia",
				Name: "Arachnophobia",
			},
			"hypochondria": {
				ID:   3,
				Tag:  "hypochondria",
				Name: "Hypochondria",
			},
			"eczema": {
				ID:   4,
				Tag:  "eczema",
				Name: "Eczema",
			},
		},
		itemCost: &common.ItemCost{
			LineItems: []*common.LineItem{
				{
					Cost: common.Cost{
						Currency: "USD",
						Amount:   4000,
					},
				},
			},
		},
		pathwayCases: []*common.PatientCase{
			{
				ID:         encoding.NewObjectID(111),
				Name:       "Acne",
				PathwayTag: "acne",
				Status:     common.PCStatusActive,
			},
			{
				ID:         encoding.NewObjectID(222),
				Name:       "Arachnophobia",
				PathwayTag: "arachnophobia",
				Status:     common.PCStatusOpen,
			},
			{
				ID:         encoding.NewObjectID(333),
				Name:       "Eczema",
				PathwayTag: "eczema",
				Status:     common.PCStatusActive,
			},
		},
		pathwayDoctors: map[string][]*common.Doctor{
			"acne": []*common.Doctor{
				{},
			},
		},
		careTeams: map[int64]*common.PatientCareTeam{
			111: &common.PatientCareTeam{
				Assignments: []*common.CareProviderAssignment{
					{
						ProviderRole:     api.DOCTOR_ROLE,
						ShortDisplayName: "Dr. Jones",
					},
				},
			},
		},
	}
	h := NewPathwayDetailsHandler(dataAPI, "api.spruce.local")

	// Unauthenticated

	r, err := http.NewRequest("GET", "/?pathway_id=acne,arachnophobia,hypochondria,eczema", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 got %d", w.Code)
	}
	res := &pathwayDetailsRes{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if len(res.Pathways) != 4 {
		t.Fatalf("Expected 4 pathways, got %d", len(res.Pathways))
	}
	for _, p := range res.Pathways {
		switch p.PathwayTag {
		default:
			t.Fatalf("Unepxected pathway tag %s", p.PathwayTag)
		case "acne":
			if p.Screen.Type != "merchandising" {
				t.Fatal("Expected acne pathway screen type to be merchandising")
			} else if p.FAQ == nil || len(p.FAQ.Views) == 0 {
				t.Fatalf("Expected acne patchway to have an FAQ: %+v", p.FAQ)
			}
		case "arachnophobia":
			if p.Screen.Type != "generic_message" {
				t.Fatal("Expected arachnophobia pathway screen type to be generic_message")
			}
		case "hypochondria":
			if p.Screen.Type != "generic_message" {
				t.Fatal("Expected hypochondria pathway screen type to be generic_message")
			}
		case "eczema":
			if p.Screen.Type != "generic_message" {
				t.Fatalf("Expected eczema screen type to be generic_message")
			}
		}

	}

	// Authenticated

	r, err = http.NewRequest("GET", "/?pathway_id=acne,arachnophobia,hypochondria,eczema", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := apiservice.GetContext(r)
	ctx.AccountID = 1
	ctx.Role = api.PATIENT_ROLE
	defer context.Clear(r)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 got %d", w.Code)
	}
	res = &pathwayDetailsRes{}
	if err := json.NewDecoder(w.Body).Decode(res); err != nil {
		t.Fatal(err)
	}
	if len(res.Pathways) != 4 {
		t.Fatalf("Expected 4 pathways, got %d", len(res.Pathways))
	}
	for _, p := range res.Pathways {
		switch p.PathwayTag {
		default:
			t.Fatalf("Unepxected pathway tag %s", p.PathwayTag)
		case "acne":
			if p.Screen.Type != "generic_message" {
				t.Fatal("Expected pathway 1 screen type to be generic_message")
			}
			if !strings.Contains(p.Screen.ContentText, "an existing") {
				t.Fatalf("Expected an existing active case message, got '%s'", p.Screen.ContentText)
			}
		case "arachnophobia":
			if p.Screen.Type != "generic_message" {
				t.Fatal("Expected arachnophobia pathway screen type to be generic_message")
			}
			if !strings.Contains(p.Screen.ContentText, "visit in progress") {
				t.Fatalf("Expected an open visit message, got '%s'", p.Screen.ContentText)
			}
		case "hypochondria":
			if p.Screen.Type != "generic_message" {
				t.Fatal("Expected pathway 2 screen type to be generic_message")
			}
		case "eczema":
			if p.Screen.Type != "generic_message" {
				t.Fatal("Expected pathway screen type to be generic_message")
			}
			if !strings.Contains(p.Screen.ContentText, "pending review") {
				t.Fatalf("Expected a pending review message, got '%s'", p.Screen.ContentText)
			}
		}
	}

	//

}

func TestParseIDList(t *testing.T) {
	testCases := map[string][]int64{
		"":      nil,
		"123":   []int64{123},
		"11,22": []int64{11, 22},
	}
	for s, expIDs := range testCases {
		ids, err := parseIDList(s)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ids, expIDs) {
			t.Fatalf("parseIDList('%s') = %+v. Expected %+v", s, ids, expIDs)
		}
	}
}
