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
	"answers": [{
		"type": "q_type_photo_section",
		"name": "Location 1",
		"photos": [{
			"creation_date": "2015-06-05T20:51:27.895527Z",
			"photo_url": "https://dev-2-api.carefront.net/v1/media?expires=1433710304\u0026media_id=6761\u0026sig=9jsrnEFuXO5QSEj6ld7qz9ACIUE%3D",
			"photo_id": "6761",
			"slot_id": "7452",
			"name": "Other Location"
		}, {
			"creation_date": "2015-06-05T20:51:27.896965Z",
			"photo_url": "https://dev-2-api.carefront.net/v1/media?expires=1433710304\u0026media_id=6763\u0026sig=n7jI7qb7Lgnj1Q9ZATCLvv-NvdU%3D",
			"photo_id": "6763",
			"slot_id": "7452",
			"name": "Other Location"
		}],
		"creation_date": "2015-06-05T20:51:27.893734Z"
	}, {
		"name": "Location 2",
		"type": "q_type_photo_section",
		"photos": [{
			"creation_date": "2015-06-05T20:51:27.899108Z",
			"photo_url": "https://dev-2-api.carefront.net/v1/media?expires=1433710304\u0026media_id=6762\u0026sig=HlShtvCWzE0wr_dbXR9RJwIlJ5A%3D",
			"photo_id": "6762",
			"slot_id": "7452",
			"name": "Other Location"
		}],
		"creation_date": "2015-06-05T20:51:27.897743Z"
	}]
	}`

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(clientJSON), &data); err != nil {
		t.Fatal(err)
	}

	var psa photoSectionAnswer
	if err := psa.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 2, len(psa.Sections))
	test.Equals(t, "Location 1", psa.Sections[0].Name)
	test.Equals(t, 2, len(psa.Sections[0].Photos))

	test.Equals(t, "6761", psa.Sections[0].Photos[0].ServerPhotoID)
	test.Equals(t, "7452", psa.Sections[0].Photos[0].SlotID)
	test.Equals(t, "Other Location", psa.Sections[0].Photos[0].Name)
	test.Equals(t, true, psa.Sections[0].Photos[0].URL != "")

	test.Equals(t, "Location 2", psa.Sections[1].Name)
	test.Equals(t, 1, len(psa.Sections[1].Photos))
}

func TestPhotoSectionAnswer_unmarshalProtobuf(t *testing.T) {
	pb := &intake.PhotoSectionPatientAnswer{
		Entries: []*intake.PhotoSectionPatientAnswer_PhotoSectionEntry{
			{
				Name: proto.String("Test"),
				Photos: []*intake.PhotoSectionPatientAnswer_PhotoSectionEntry_PhotoSlotAnswer{
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

	var psa photoSectionAnswer
	if err := psa.unmarshalProtobuf(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 1, len(psa.Sections))
	test.Equals(t, "Test", psa.Sections[0].Name)

	test.Equals(t, 2, len(psa.Sections[0].Photos))
	test.Equals(t, "SlotName", psa.Sections[0].Photos[0].Name)
	test.Equals(t, "123", psa.Sections[0].Photos[0].SlotID)
	test.Equals(t, "12", psa.Sections[0].Photos[0].LocalPhotoID)
	test.Equals(t, "12345", psa.Sections[0].Photos[0].URL)

	test.Equals(t, "SlotName2", psa.Sections[0].Photos[1].Name)
	test.Equals(t, "124", psa.Sections[0].Photos[1].SlotID)
	test.Equals(t, "13", psa.Sections[0].Photos[1].ServerPhotoID)
}

func TestPhotoSectionAnswer_transformToProtobuf(t *testing.T) {
	psa := &photoSectionAnswer{
		Sections: []*photoSectionAnswerItem{
			{
				Name: "Section1",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
					},
				},
			},
		},
	}

	pb, err := psa.transformToProtobuf()
	if err != nil {
		t.Fatal(err)
	}

	ps, ok := pb.(*intake.PhotoSectionPatientAnswer)
	test.Equals(t, true, ok)
	test.Equals(t, 1, len(ps.Entries))
	test.Equals(t, 2, len(ps.Entries[0].Photos))
	test.Equals(t, "12", *ps.Entries[0].Photos[0].SlotId)
	test.Equals(t, "13", *ps.Entries[0].Photos[1].SlotId)
	test.Equals(t, "12", *ps.Entries[0].Photos[0].Id.Id)
	test.Equals(t, "13", *ps.Entries[0].Photos[1].Id.Id)
}

func TestPhotoSectionAnswer_marshalJSONForClient(t *testing.T) {
	expectedJSON := `{"question_id":"10","answered_photo_sections":[{"name":"Section1","photos":[{"name":"Slot1","slot_id":"12","photo_id":"12"},{"name":"Slot2","slot_id":"13","photo_id":"13"}]},{"name":"Section2","photos":[{"name":"Slot1","slot_id":"12","photo_id":"12"},{"name":"Slot2","slot_id":"13","photo_id":"13"}]}]}`
	psa := &photoSectionAnswer{
		QuestionID: "10",
		Sections: []*photoSectionAnswerItem{
			{
				Name: "Section1",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
					},
				},
			},
			{
				Name: "Section2",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
					},
				},
			},
		},
	}

	jsonData, err := psa.marshalJSONForClient()
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, expectedJSON, string(jsonData))
}

func TestPhotoSectionAnswer_equals(t *testing.T) {
	psa := &photoSectionAnswer{
		QuestionID: "10",
		Sections: []*photoSectionAnswerItem{
			{
				Name: "Section1",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
					},
				},
			},
			{
				Name: "Section2",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
					},
				},
			},
		},
	}

	if !psa.equals(psa) {
		t.Fatal("expected answers to be equal")
	}

	// answers should not be equal when there is a mismatched number of photos
	other := &photoSectionAnswer{
		QuestionID: "10",
		Sections: []*photoSectionAnswerItem{
			{
				Name: "Section1",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
				},
			},
			{
				Name: "Section2",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
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
	other = &photoSectionAnswer{
		QuestionID: "10",
		Sections: []*photoSectionAnswerItem{
			{
				Name: "Section1",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
						URL:           "sup",
					},
				},
			},
			{
				Name: "Section2",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
					},
				},
			},
		},
	}

	if !psa.equals(other) {
		t.Fatal("expected answers to be equal even when URLs don't match")
	}

	// answers should not be equal when the section names don't match
	other = &photoSectionAnswer{
		QuestionID: "10",
		Sections: []*photoSectionAnswerItem{
			{
				Name: "Section12",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
					},
				},
			},
			{
				Name: "Section2",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
					},
				},
			},
		},
	}

	if psa.equals(other) {
		t.Fatal("expected answers to not be equal")
	}

	// answers shoult not be equal when the photo names don't match
	other = &photoSectionAnswer{
		QuestionID: "10",
		Sections: []*photoSectionAnswerItem{
			{
				Name: "Section1",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot21",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
					},
				},
			},
			{
				Name: "Section2",
				Photos: []*photoSlotAnswerItem{
					{
						Name:          "Slot1",
						SlotID:        "12",
						ServerPhotoID: "12",
					},
					{
						Name:          "Slot2",
						SlotID:        "13",
						ServerPhotoID: "13",
					},
				},
			},
		},
	}

	if psa.equals(other) {
		t.Fatal("expected answers to not be equal")
	}
}
