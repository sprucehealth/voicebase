package manager

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

var genericPopupJSON = `
{
  "screen_type": "screen_type_generic_popup",
  "type": "screen_type_generic_popup",
  "condition": {
    "op": "answer_contains_any",
    "type": "answer_contains_any",
    "question": "q_derm_tick_bite_tick_still_attached",
    "question_id": "44812",
    "potential_answers_id": [
      "148434"
    ],
    "potential_answers": [
      "a_derm_tick_bite_tick_still_attached_yes"
    ]
  },
  "client_data": {
    "views": [
      {
        "element_style": "numbered",
        "number": 1,
        "text": "Find a pair of blunt forceps or tweezers. If you don’t have those, use a rubber glove or paper towel.",
        "type": "treatment:list_element"
      },
      {
        "element_style": "numbered",
        "number": 2,
        "text": "Grasp the tick as close to the skin as possible. Do not crush or twist the tick.",
        "type": "treatment:list_element"
      },
      {
        "element_style": "numbered",
        "number": 3,
        "text": "Pull upwards gently and steadily until the tick is removed from your skin.",
        "type": "treatment:list_element"
      },
      {
        "element_style": "numbered",
        "number": 4,
        "text": "Place the tick in a jar or other safe place. We’ll need to take a photo of it for your doctor.",
        "type": "treatment:list_element"
      },
      {
        "element_style": "numbered",
        "number": 5,
        "text": "Wash your hands and the tick bite thoroughly with soap and water.",
        "type": "treatment:list_element"
      }
    ]
  }
}`

func TestScreenGenericPopup_Parsing(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(genericPopupJSON), &data); err != nil {
		t.Fatal(err)
	}

	gs := &genericPopupScreen{}
	if err := gs.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, true, gs.ViewDataJSON != nil)
}

func TestScreenGenericPopup_staticInfoCopy(t *testing.T) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(genericPopupJSON), &data); err != nil {
		t.Fatal(err)
	}

	gs := &genericPopupScreen{}
	if err := gs.unmarshalMapFromClient(data, nil, &visitManager{}); err != nil {
		t.Fatal(err)
	}

	gs2 := gs.staticInfoCopy(nil).(*genericPopupScreen)
	test.Equals(t, true, &gs.ViewDataJSON != &gs2.ViewDataJSON)
	test.Equals(t, gs.ViewDataJSON, gs2.ViewDataJSON)
}
