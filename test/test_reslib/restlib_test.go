package test_reslib

import (
	"testing"

	"github.com/sprucehealth/backend/apiclient"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestResourceGuide(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)
	testData.StartAPIServer(t)
	sec1 := common.ResourceGuideSection{
		Title:   "Section 1",
		Ordinal: 1,
	}
	if _, err := testData.DataAPI.CreateResourceGuideSection(&sec1); err != nil {
		t.Fatal(err)
	}

	guide1 := common.ResourceGuide{
		SectionID: sec1.ID,
		Ordinal:   1,
		Title:     "Guide 1",
		PhotoURL:  "http://example.com/1.jpeg",
		Layout:    "noop",
		Tag:       "tag1",
		Active:    true,
	}
	if _, err := testData.DataAPI.CreateResourceGuide(&guide1); err != nil {
		t.Fatal(err)
	}
	guide2 := common.ResourceGuide{
		SectionID: sec1.ID,
		Ordinal:   2,
		Title:     "Guide 1",
		PhotoURL:  "http://example.com/1.jpeg",
		Layout:    "noop",
		Tag:       "tag2",
		Active:    true,
	}
	if _, err := testData.DataAPI.CreateResourceGuide(&guide2); err != nil {
		t.Fatal(err)
	}

	cli := &apiclient.PatientClient{
		Config: apiclient.Config{
			BaseURL: testData.APIServer.URL,
		},
	}

	guide, err := cli.ResourceGuide(guide1.ID)
	test.OK(t, err)
	test.Equals(t, "noop", guide)

	sections, err := cli.ListResourceGuides()
	test.OK(t, err)
	test.Equals(t, 1, len(sections))
	test.Equals(t, 2, len(sections[0].Guides))
}
