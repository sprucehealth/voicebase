package manager

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

var screenWarningPopupJSON = `
 {
	"screen_type": "screen_type_warning_popup",
	"type": "screen_type_warning_popup",
	"condition": {
		"op": "answer_contains_any",
		"type": "answer_contains_any",
		"question": "q_derm_rash_had_this_rash_before",
		"question_id": "40546",
		"potential_answers_id": ["126291"],
		"potential_answers": ["a_derm_rash_had_this_rash_before_never"]
	},
	"body": {
		"text": "Your answers suggest that your doctor..."
	},
	"bottom_button_title": "Next Steps",
	"content_header_title": "We recommend you start a new \"Rash\" visit"
}`

func TestScreenWarningPopupParsing(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(screenWarningPopupJSON), &data); err != nil {
		t.Fatal(err)
	}

	ws := &warningPopupScreen{}
	if err := ws.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, "We recommend you start a new \"Rash\" visit", ws.ContentHeaderTitle)
	test.Equals(t, "Next Steps", ws.BottomButtonTitle)
	test.Equals(t, true, ws.Body != nil)
	test.Equals(t, "Your answers suggest that your doctor...", ws.Body.Text)
	test.Equals(t, "spruce:///image/icon_triage_alert", ws.ImageLink)

}

func TestScreenWarningPopup_staticInfoCopy(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(screenWarningPopupJSON), &data); err != nil {
		t.Fatal(err)
	}

	ws := &warningPopupScreen{}
	if err := ws.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	ws2 := ws.staticInfoCopy(nil).(*warningPopupScreen)

	test.Equals(t, ws.Body, ws2.Body)
}
