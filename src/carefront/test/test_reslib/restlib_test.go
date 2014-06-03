package test_reslib

import (
	"carefront/common"
	"carefront/reslib"
	"carefront/test/test_integration"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResourceGuide(t *testing.T) {
	testData := test_integration.SetupIntegrationTest(t)
	defer test_integration.TearDownIntegrationTest(t, testData)

	guide1 := common.ResourceGuide{
		Title:    "Guide 1",
		PhotoURL: "http://example.com/1.jpeg",
		Layout:   "noop",
	}
	if _, err := testData.DataApi.CreateResourceGuide(&guide1); err != nil {
		t.Fatal(err)
	}
	guide2 := common.ResourceGuide{
		Title:    "Guide 1",
		PhotoURL: "http://example.com/1.jpeg",
		Layout:   "noop",
	}
	if _, err := testData.DataApi.CreateResourceGuide(&guide2); err != nil {
		t.Fatal(err)
	}

	h := reslib.NewHandler(testData.DataApi)
	hList := reslib.NewListHandler(testData.DataApi)

	res := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/?resource_id=%d", guide1.Id), nil)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Fatalf("Expected 200 response got %d", res.Code)
	}
	var v string
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		t.Fatal(err)
	} else if v != "noop" {
		t.Fatalf("Layout does not match. Expected 'noop' got '%s'", v)
	}

	res = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	hList.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Fatalf("Expected 200 response got %d", res.Code)
	}
	var lr reslib.ListResponse
	if err := json.NewDecoder(res.Body).Decode(&lr); err != nil {
		t.Fatal(err)
	} else if len(lr.Guides) != 2 {
		t.Fatalf("Expected 2 guides. Got %d", len(lr.Guides))
	}
}
