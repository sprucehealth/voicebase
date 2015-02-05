package patient_visit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
)

func reindentJson(b []byte) ([]byte, error) {
	var js interface{}
	if err := json.Unmarshal(b, &js); err != nil {
		return nil, err
	}
	return json.MarshalIndent(js, "", "  ")
}

type pathwayMenuHandlerDataAPI struct {
	api.DataAPI
	patient           *common.Patient
	hasSubmittedVisit bool
	pathwayMenu       *common.PathwayMenu
}

func (api *pathwayMenuHandlerDataAPI) GetPatientFromAccountID(accountID int64) (*common.Patient, error) {
	return api.patient, nil
}

func (api *pathwayMenuHandlerDataAPI) AnyVisitSubmitted(patientID int64) (bool, error) {
	return api.hasSubmittedVisit, nil
}

func (api *pathwayMenuHandlerDataAPI) PathwayMenu() (*common.PathwayMenu, error) {
	return api.pathwayMenu, nil
}

func TestPathwayMenuHandler(t *testing.T) {
	dataAPI := &pathwayMenuHandlerDataAPI{
		patient: &common.Patient{
			StateFromZipCode: "CA",
			Gender:           "female",
			DOB: encoding.DOB{
				Day:   13,
				Month: 6,
				Year:  1999,
			},
		},
		hasSubmittedVisit: false,
		pathwayMenu: &common.PathwayMenu{
			Title: "What are you here to see the doctor for today?",
			Items: []*common.PathwayMenuItem{
				{
					Title:      "Acne",
					Type:       common.PathwayMenuItemTypePathway,
					PathwayTag: "acne",
				},
				{
					Title: "Anti-aging",
					Type:  common.PathwayMenuItemTypeMenu,
					Menu: &common.PathwayMenu{
						Title: "Getting old? What would you like to see the doctor for?",
						Items: []*common.PathwayMenuItem{
							{
								Title:      "Wrinkles",
								Type:       common.PathwayMenuItemTypePathway,
								PathwayTag: "wrinkles",
							},
							{
								Title: "Hair Loss",
								Type:  common.PathwayMenuItemTypePathway,
								Conditionals: []*common.Conditional{
									{
										Op:    "==",
										Key:   "gender",
										Value: "male",
									},
								},
								PathwayTag: "hairloss",
							},
						},
					},
				},
			},
		},
	}
	h := NewPathwayMenuHandler(dataAPI)

	// Unauthenticated

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	wr := httptest.NewRecorder()
	h.ServeHTTP(wr, r)
	if wr.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", wr.Code, wr.Body.String())
	}
	js, err := reindentJson(wr.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	exp := `{
  "data": {
    "children": [
      {
        "data": {
          "id": "acne"
        },
        "title": "Acne",
        "type": "pathway"
      },
      {
        "data": {
          "children": [
            {
              "data": {
                "id": "wrinkles"
              },
              "title": "Wrinkles",
              "type": "pathway"
            }
          ],
          "title": "Getting old? What would you like to see the doctor for?"
        },
        "title": "Anti-aging",
        "type": "container"
      }
    ],
    "title": "What are you here to see the doctor for today?"
  },
  "title": "",
  "type": "container"
}`
	if string(js) != exp {
		t.Fatalf("\nExpected %s\ngot %s", exp, string(js))
	}

	// Authenticated

	r, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := apiservice.GetContext(r)
	ctx.AccountID = 1
	ctx.Role = api.PATIENT_ROLE
	defer context.Clear(r)
	wr = httptest.NewRecorder()
	h.ServeHTTP(wr, r)
	if wr.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", wr.Code, wr.Body.String())
	}
	js, err = reindentJson(wr.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	// Should have filtered out the hair loss pathway
	exp = `{
  "data": {
    "bottom_button_tap_url": "spruce:///action/view_home",
    "bottom_button_title": "Not ready to start a visit yet?",
    "children": [
      {
        "data": {
          "id": "acne"
        },
        "title": "Acne",
        "type": "pathway"
      },
      {
        "data": {
          "children": [
            {
              "data": {
                "id": "wrinkles"
              },
              "title": "Wrinkles",
              "type": "pathway"
            }
          ],
          "title": "Getting old? What would you like to see the doctor for?"
        },
        "title": "Anti-aging",
        "type": "container"
      }
    ],
    "title": "What are you here to see the doctor for today?"
  },
  "title": "",
  "type": "container"
}`
	if string(js) != exp {
		t.Fatalf("\nExpected %s\ngot %s", exp, string(js))
	}
}

func TestMatchesConditionals(t *testing.T) {
	cond := []*common.Conditional{
		{Op: "==", Key: "gender", Value: "male"},
		{Op: "==", Key: "state", Value: "CA", Not: true},
	}
	ctx := map[string]interface{}{
		"gender": "male",
		"state":  "IL",
	}
	if b, err := matchesConditionals(ctx, cond); err != nil {
		t.Fatal(err)
	} else if !b {
		t.Fatalf("Expected to match but didn't")
	}
	ctx = map[string]interface{}{
		"gender": "male",
		"state":  "CA",
	}
	if b, err := matchesConditionals(ctx, cond); err != nil {
		t.Fatal(err)
	} else if b {
		t.Fatalf("Expected to not match but did")
	}
	ctx = map[string]interface{}{
		"gender": "female",
		"state":  "IL",
	}
	if b, err := matchesConditionals(ctx, cond); err != nil {
		t.Fatal(err)
	} else if b {
		t.Fatalf("Expected to not match but did")
	}
}

func TestConditionalIsEqual(t *testing.T) {
	// Strings
	if b, err := isEqual("a", "a"); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected "a" == "a" to be true`)
	}
	if b, err := isEqual("a", "b"); err != nil {
		t.Error(err)
	} else if b {
		t.Errorf(`Expected "a" == "b" to be false`)
	}

	// Numbers
	if b, err := isEqual(1, 1); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected 1 == 1 to be true`)
	}
	if b, err := isEqual(1, 2); err != nil {
		t.Error(err)
	} else if b {
		t.Errorf(`Expected 1 == 1 to be false`)
	}
	if b, err := isEqual(1, float64(1.0)); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected 1 == 1.0 to be true`)
	}
}

func TestConditionalIsLessThan(t *testing.T) {
	// Strings
	if b, err := isLessThan("a", "a"); err != nil {
		t.Error(err)
	} else if b {
		t.Errorf(`Expected "a" < "a" to be false`)
	}
	if b, err := isLessThan("a", "b"); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected "a" < "b" to be true`)
	}

	// Numbers
	if b, err := isLessThan(1, 1); err != nil {
		t.Error(err)
	} else if b {
		t.Errorf(`Expected 1 < 1 to be false`)
	}
	if b, err := isLessThan(1, 2); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected 1 < 2 to be true`)
	}
	if b, err := isLessThan(1, float64(2.0)); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected 1 < 2.0 to be true`)
	}
}

func TestConditionalIsGreaterThan(t *testing.T) {
	// Strings
	if b, err := isGreaterThan("a", "a"); err != nil {
		t.Error(err)
	} else if b {
		t.Errorf(`Expected "a" > "a" to be false`)
	}
	if b, err := isGreaterThan("b", "a"); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected "b" > "a" to be true`)
	}

	// Numbers
	if b, err := isGreaterThan(1, 1); err != nil {
		t.Error(err)
	} else if b {
		t.Errorf(`Expected 1 > 1 to be false`)
	}
	if b, err := isGreaterThan(2, 1); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected 2 > 1 to be true`)
	}
	if b, err := isGreaterThan(2, float64(1.0)); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected 2 > 1.0 to be true`)
	}
}

func TestConditionalIsIn(t *testing.T) {
	// Strings
	if b, err := isIn("a", []string{"b", "c"}); err != nil {
		t.Error(err)
	} else if b {
		t.Errorf(`Expected "a" in ["b", "c"] to be false`)
	}
	if b, err := isIn("b", []string{"b", "c"}); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected "b" in ["b", "b"] to be true`)
	}

	// Numbers
	if b, err := isIn(1, []int{2, 3}); err != nil {
		t.Error(err)
	} else if b {
		t.Errorf(`Expected 1 in [2, 3] to be false`)
	}
	if b, err := isIn(2, []int{2, 3}); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected 2 in [2, 3 to be true`)
	}
	if b, err := isIn(2, []float64{2.0, 3.0}); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected 2 in [2.0, 3.0] to be true`)
	}

	// Interface slices

	// Strings
	if b, err := isIn("a", []interface{}{"b", "c"}); err != nil {
		t.Error(err)
	} else if b {
		t.Errorf(`Expected "a" in ["b", "c"] to be false`)
	}
	if b, err := isIn("b", []interface{}{"b", "c"}); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected "b" in ["b", "b"] to be true`)
	}

	// Numbers
	if b, err := isIn(1, []interface{}{2, 3}); err != nil {
		t.Error(err)
	} else if b {
		t.Errorf(`Expected 1 in [2, 3] to be false`)
	}
	if b, err := isIn(2, []interface{}{2, 3}); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected 2 in [2, 3 to be true`)
	}
	if b, err := isIn(2, []interface{}{2.0, 3.0}); err != nil {
		t.Error(err)
	} else if !b {
		t.Errorf(`Expected 2 in [2.0, 3.0] to be true`)
	}
}
