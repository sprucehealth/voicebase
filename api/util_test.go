package api

import (
	"testing"

	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_SectionTest struct {
	called bool
	DataAPI
}

func (m *mockDataAPI_SectionTest) GetSectionInfo(sectionTag string, languageID int64) (int64, string, error) {
	m.called = true
	return 1, "", nil
}

// this test is to ensure that if the section details
// exist in the layout itself then there is no need to look them up
// in the database
func TestSection_ExistsInLayout(t *testing.T) {
	m := &mockDataAPI_SectionTest{}
	test.OK(t, fillSection(&info_intake.Section{
		SectionTag:   "test",
		SectionTitle: "123",
		SectionId:    "test",
	}, m, EN_LANGUAGE_ID))
	test.Equals(t, false, m.called)
}

// this test is to ensure that if the section details do
// not adequately exist in the layout then we need to look them up in
// the database
func TestSection_DoesNotExistInLayout(t *testing.T) {
	m := &mockDataAPI_SectionTest{}
	test.OK(t, fillSection(&info_intake.Section{
		SectionTag: "test",
	}, m, EN_LANGUAGE_ID))
	test.Equals(t, true, m.called)
}
