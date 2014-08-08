package test_reslib

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/reslib"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestResourceGuide(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	sec1 := common.ResourceGuideSection{
		Title:   "Section 1",
		Ordinal: 1,
	}
	if _, err := testData.DataApi.CreateResourceGuideSection(&sec1); err != nil {
		t.Fatal(err)
	}

	guide1 := common.ResourceGuide{
		SectionID: sec1.ID,
		Ordinal:   1,
		Title:     "Guide 1",
		PhotoURL:  "http://example.com/1.jpeg",
		Layout:    "noop",
	}
	if _, err := testData.DataApi.CreateResourceGuide(&guide1); err != nil {
		t.Fatal(err)
	}
	guide2 := common.ResourceGuide{
		SectionID: sec1.ID,
		Ordinal:   2,
		Title:     "Guide 1",
		PhotoURL:  "http://example.com/1.jpeg",
		Layout:    "noop",
	}
	if _, err := testData.DataApi.CreateResourceGuide(&guide2); err != nil {
		t.Fatal(err)
	}

	h := reslib.NewHandler(testData.DataApi)
	hList := reslib.NewListHandler(testData.DataApi)

	res := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/?resource_id=%d", guide1.ID), nil)
	test.OK(t, err)
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
	test.OK(t, err)
	hList.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Fatalf("Expected 200 response got %d", res.Code)
	}
	var lr reslib.ListResponse
	if err := json.NewDecoder(res.Body).Decode(&lr); err != nil {
		t.Fatal(err)
	} else if len(lr.Sections) != 1 {
		t.Fatalf("Expected 1 section. Got %d", len(lr.Sections))
	} else if len(lr.Sections[0].Guides) != 2 {
		t.Fatalf("Expected 2 guides. Got %d", len(lr.Sections[0].Guides))
	}
}
