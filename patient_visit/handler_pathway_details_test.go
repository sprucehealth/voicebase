package patient_visit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/sku"
)

type pathwayDetailsHandlerDataAPI struct {
	api.DataAPI
	pathways       map[int64]*common.Pathway
	pathwayCases   map[int64]int64
	pathwayDoctors map[int64][]*common.Doctor
	careTeams      map[int64]*common.PatientCareTeam
	itemCost       *common.ItemCost
}

// pathwayDetailsRes is a simplified response version of the pathway details handler response.
// It's not possible to use the existing response struct because it uses interfaces.
type pathwayDetailsRes struct {
	Pathways []struct {
		ID     int64 `json:"pathway_id,string"`
		Screen struct {
			Type string `json:"type"`
		} `json:"screen"`
	} `json:"pathway_details_screens"`
}

func (api *pathwayDetailsHandlerDataAPI) GetPatientIDFromAccountID(accountID int64) (int64, error) {
	return 1, nil
}

func (api *pathwayDetailsHandlerDataAPI) ActiveCaseIDsForPathways(patientID int64) (map[int64]int64, error) {
	return api.pathwayCases, nil
}

func (api *pathwayDetailsHandlerDataAPI) Pathways(ids []int64, opts api.PathwayOption) (map[int64]*common.Pathway, error) {
	ps := make(map[int64]*common.Pathway, len(ids))
	for _, id := range ids {
		if p := api.pathways[id]; p != nil {
			ps[id] = p
		}
	}
	return ps, nil
}

func (api *pathwayDetailsHandlerDataAPI) GetCareTeamsForPatientByCase(patientID int64) (map[int64]*common.PatientCareTeam, error) {
	return api.careTeams, nil
}

func (api *pathwayDetailsHandlerDataAPI) DoctorsForPathway(pathwayID int64, limit int) ([]*common.Doctor, error) {
	return api.pathwayDoctors[pathwayID], nil
}

func (api *pathwayDetailsHandlerDataAPI) GetActiveItemCost(itemType sku.SKU) (*common.ItemCost, error) {
	return api.itemCost, nil
}

func TestPathwayDetailsHandler(t *testing.T) {
	dataAPI := &pathwayDetailsHandlerDataAPI{
		pathways: map[int64]*common.Pathway{
			1: {
				ID:   1,
				Name: "Acne",
				Details: &common.PathwayDetails{
					WhatIsIncluded: []string{"Cheese"},
					WhoWillTreatMe: "George Carlin",
					RightForMe:     "Probably",
					DidYouKnow:     []string{"BEEEEES"},
				},
			},
			2: {
				ID:   2,
				Name: "Hypochondria",
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
		pathwayCases: map[int64]int64{
			1: 123,
		},
		pathwayDoctors: map[int64][]*common.Doctor{
			1: []*common.Doctor{
				{
					LargeThumbnailURL: "http://example.com/image.jpeg",
				},
			},
		},
		careTeams: map[int64]*common.PatientCareTeam{
			123: &common.PatientCareTeam{
				Assignments: []*common.CareProviderAssignment{
					{
						ProviderRole:     api.DOCTOR_ROLE,
						ShortDisplayName: "Dr. Jones",
					},
				},
			},
		},
	}
	h := NewPathwayDetailsHandler(dataAPI)

	// Unauthenticated

	r, err := http.NewRequest("GET", "/?pathway_id=1,2", nil)
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
	if len(res.Pathways) != 2 {
		t.Fatalf("Expected 2 pathways, got %d", len(res.Pathways))
	}
	for _, p := range res.Pathways {
		if p.ID == 1 && p.Screen.Type != "merchandising" {
			t.Fatal("Expected pathway 1 screen type to be merchandising")
		} else if p.ID == 2 && p.Screen.Type != "generic_message" {
			t.Fatal("Expected pathway 2 screen type to be generic_message")
		}
	}

	// Authenticated

	r, err = http.NewRequest("GET", "/?pathway_id=1,2", nil)
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
	if len(res.Pathways) != 2 {
		t.Fatalf("Expected 2 pathways, got %d", len(res.Pathways))
	}
	for _, p := range res.Pathways {
		if p.ID == 1 && p.Screen.Type != "generic_message" {
			t.Fatal("Expected pathway 1 screen type to be generic_message")
		} else if p.ID == 2 && p.Screen.Type != "generic_message" {
			t.Fatal("Expected pathway 2 screen type to be generic_message")
		}
	}
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
