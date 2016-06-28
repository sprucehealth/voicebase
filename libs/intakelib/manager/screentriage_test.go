package manager

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

const screenTriageJSON = `
{
	"screen_type": "screen_type_triage",
	"type": "screen_type_triage",
	"condition": {
		"op": "answer_contains_any",
		"type": "answer_contains_any",
		"question": "q_derm_rash_type_of_healthcare_provider_previously_seen",
		"question_id": "40601",
		"potential_answers_id": ["126469"],
		"potential_answers": ["a_derm_rash_type_of_healthcare_provider_previously_seen_i_was_hospitalized_for_it"]
	},
	"body": {
		"text": "If you have health insurance,..."
	},
	"bottom_button_title": "I Understand",
	"content_header_title": "You should seek in-person medical evaluation today. A local urgent care is an appropriate option, as is your primary care provider.",
	"screen_title": "Next Steps",
	"client_data": {
					"pathway_id": "derm_rash",
					"triage_parameters": {
						"abandon": true
					},
					"views": null
     }
}`

func TestScreenTriageParsing(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(screenTriageJSON), &data); err != nil {
		t.Fatal(err)
	}

	ts := &triageScreen{}
	if err := ts.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "You should seek in-person medical evaluation today. A local urgent care is an appropriate option, as is your primary care provider.", ts.ContentHeaderTitle)
	test.Equals(t, true, ts.Body != nil)
	test.Equals(t, "If you have health insurance,...", ts.Body.Text)
	test.Equals(t, "Next Steps", ts.Title)
	test.Equals(t, "I Understand", ts.BottomButtonTitle)
	test.Equals(t, true, ts.IsTriageScreen)
	test.Equals(t, "derm_rash", ts.TriagePathwayID)
	test.Equals(t, true, ts.TriageParametersJSON != nil)
}

func TestScreenTriage_staticInfo(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(screenTriageJSON), &data); err != nil {
		t.Fatal(err)
	}

	ts := &triageScreen{}
	if err := ts.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	ts2 := ts.staticInfoCopy(nil).(*triageScreen)
	test.Equals(t, ts, ts2)
}

type mockDataSource_screen struct {
	q question
}

func (m *mockDataSource_screen) question(questionID string) question {
	return m.q
}

func (m *mockDataSource_screen) valueForKey(key string) []byte {
	return nil
}

func (m *mockDataSource_screen) registerDependencies(layoutUnit, []layoutUnit) {}
