package manager

import (
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
	"github.com/sprucehealth/backend/libs/test"
)

func TestPhotoSectionAnswer_unmarshalMapFromClient(t *testing.T) {
	clientJSON := `{
	  "type": "q_type_media_section",
	  "sections": [
	    {
	      "name": "Location 1",
	      "media": [
	        {
	          "url": "https://dev-2-api.carefront.net/v1/media?expires=1433710304&media_id=6761&sig=9jsrnEFuXO5QSEj6ld7qz9ACIUE%3D",
						"thumbnail_url": "https://dev-2-api.carefront.net/v1/media?expires=1433710304&media_id=6761&sig=9jsrnEFuXO5QSEj6ld7qz9ACIUE%3D",
	          "media_id": "6761",
	          "slot_id": "7452",
						"type": "image",
	          "name": "Other Location"
	        },
	        {
	          "url": "https://dev-2-api.carefront.net/v1/media?expires=1433710304&media_id=6763&sig=n7jI7qb7Lgnj1Q9ZATCLvv-NvdU%3D",
						"thumbnail_url": "https://dev-2-api.carefront.net/v1/media?expires=1433710304&media_id=6761&sig=9jsrnEFuXO5QSEj6ld7qz9ACIUE%3D",
	          "media_id": "6763",
	          "slot_id": "7452",
						"type": "image",
	          "name": "Other Location"
	        }
	      ]
	    },
	    {
	      "name": "Location 2",
	      "type": "q_type_photo_section",
	      "media": [
	        {
	          "url": "https://dev-2-api.carefront.net/v1/media?expires=1433710304&media_id=6762&sig=HlShtvCWzE0wr_dbXR9RJwIlJ5A%3D",
						"thumbnail_url": "https://dev-2-api.carefront.net/v1/media?expires=1433710304&media_id=6762&sig=HlShtvCWzE0wr_dbXR9RJwIlJ5A%3D",
	          "media_id": "6762",
	          "slot_id": "7452",
						"type": "image",
	          "name": "Other Location"
	        }
	      ]
	    }
	  ]
	}`

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(clientJSON), &data); err != nil {
		t.Fatal(err)
	}

	var psa mediaSectionAnswer
	if err := psa.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 2, len(psa.Sections))
	test.Equals(t, "Location 1", psa.Sections[0].Name)
	test.Equals(t, 2, len(psa.Sections[0].Media))

	test.Equals(t, "6761", psa.Sections[0].Media[0].ServerMediaID)
	test.Equals(t, "7452", psa.Sections[0].Media[0].SlotID)
	test.Equals(t, "Other Location", psa.Sections[0].Media[0].Name)
	test.Equals(t, true, psa.Sections[0].Media[0].URL != "")

	test.Equals(t, "Location 2", psa.Sections[1].Name)
	test.Equals(t, 1, len(psa.Sections[1].Media))
}

func TestPhotoSectionAnswer_unmarshalProtobuf(t *testing.T) {
	pb := &intake.MediaSectionPatientAnswer{
		Entries: []*intake.MediaSectionPatientAnswer_MediaSectionEntry{
			{
				Name: proto.String("Test"),
				Media: []*intake.MediaSectionPatientAnswer_MediaSectionEntry_MediaSlotAnswer{
					{
						Name:   proto.String("SlotName"),
						SlotId: proto.String("123"),
						Id: &intake.ID{
							Id:   proto.String("12"),
							Type: intake.ID_LOCAL.Enum(),
						},
						Link: proto.String("12345"),
					},
					{
						Name:   proto.String("SlotName2"),
						SlotId: proto.String("124"),
						Id: &intake.ID{
							Id:   proto.String("13"),
							Type: intake.ID_SERVER.Enum(),
						},
					},
				},
			},
		},
	}

	data, err := proto.Marshal(pb)
	if err != nil {
		t.Fatal(err)
	}

	var psa mediaSectionAnswer
	if err := psa.unmarshalProtobuf(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 1, len(psa.Sections))
	test.Equals(t, "Test", psa.Sections[0].Name)

	test.Equals(t, 2, len(psa.Sections[0].Media))
	test.Equals(t, "SlotName", psa.Sections[0].Media[0].Name)
	test.Equals(t, "123", psa.Sections[0].Media[0].SlotID)
	test.Equals(t, "12", psa.Sections[0].Media[0].LocalMediaID)
	test.Equals(t, "12345", psa.Sections[0].Media[0].URL)

	test.Equals(t, "SlotName2", psa.Sections[0].Media[1].Name)
	test.Equals(t, "124", psa.Sections[0].Media[1].SlotID)
	test.Equals(t, "13", psa.Sections[0].Media[1].ServerMediaID)
}

func TestPhotoSectionAnswer_transformToProtobuf(t *testing.T) {
	psa := &mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Name: "Section1",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
					},
				},
			},
		},
	}

	pb, err := psa.transformToProtobuf()
	if err != nil {
		t.Fatal(err)
	}

	ps, ok := pb.(*intake.MediaSectionPatientAnswer)
	test.Equals(t, true, ok)
	test.Equals(t, 1, len(ps.Entries))
	test.Equals(t, 2, len(ps.Entries[0].Media))
	test.Equals(t, "12", *ps.Entries[0].Media[0].SlotId)
	test.Equals(t, "13", *ps.Entries[0].Media[1].SlotId)
	test.Equals(t, "12", *ps.Entries[0].Media[0].Id.Id)
	test.Equals(t, "13", *ps.Entries[0].Media[1].Id.Id)
}

func TestPhotoSectionAnswer_transformForClient(t *testing.T) {
	expectedJSON := `{"type":"q_type_media_section","sections":[{"name":"Section1","media":[{"name":"Slot1","slot_id":"12","media_id":"12","type":"image"},{"name":"Slot2","slot_id":"13","media_id":"13","type":"image"}]},{"name":"Section2","media":[{"name":"Slot1","slot_id":"12","media_id":"12","type":"image"},{"name":"Slot2","slot_id":"13","media_id":"13","type":"image"}]}]}`
	psa := &mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Name: "Section1",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
						Type:          "image",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
						Type:          "image",
					},
				},
			},
			{
				Name: "Section2",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
						Type:          "image",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
						Type:          "image",
					},
				},
			},
		},
	}

	data, err := psa.transformForClient()
	if err != nil {
		t.Fatal(err)
	}

	jsonData, err := json.Marshal(data)
	test.OK(t, err)

	test.Equals(t, expectedJSON, string(jsonData))
}

func TestPhotoSectionAnswer_equals(t *testing.T) {
	psa := &mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Name: "Section1",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
					},
				},
			},
			{
				Name: "Section2",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
					},
				},
			},
		},
	}

	if !psa.equals(psa) {
		t.Fatal("expected answers to be equal")
	}

	// answers should not be equal when there is a mismatched number of photos
	other := &mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Name: "Section1",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
					},
				},
			},
			{
				Name: "Section2",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
					},
				},
			},
		},
	}

	if psa.equals(other) {
		t.Fatal("expected answers to not be equal")
	}

	// answers should be equal even if one has a URL while the other doesn't
	// because the client would not necessarily send the URL to the library
	other = &mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Name: "Section1",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
						URL:           "sup",
					},
				},
			},
			{
				Name: "Section2",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
					},
				},
			},
		},
	}

	if !psa.equals(other) {
		t.Fatal("expected answers to be equal even when URLs don't match")
	}

	// answers should not be equal when the section names don't match
	other = &mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Name: "Section12",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
					},
				},
			},
			{
				Name: "Section2",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
					},
				},
			},
		},
	}

	if psa.equals(other) {
		t.Fatal("expected answers to not be equal")
	}

	// answers shoult not be equal when the photo names don't match
	other = &mediaSectionAnswer{
		Sections: []*mediaSectionAnswerItem{
			{
				Name: "Section1",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot21",
						SlotID:        "12",
						ServerMediaID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
					},
				},
			},
			{
				Name: "Section2",
				Media: []*mediaSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerMediaID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerMediaID: "13",
					},
				},
			},
		},
	}

	if psa.equals(other) {
		t.Fatal("expected answers to not be equal")
	}
}
