package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestUpdateEntityMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "a_1",
	}
	ctx = ctxWithAccount(ctx, acc)

	entityID := "e_1"

	entityInfo := &directory.EntityInfo{
		FirstName:     "firstName",
		MiddleInitial: "middleInitial",
		LastName:      "lastName",
		GroupName:     "groupName",
		DisplayName:   "firstName middleInitial. lastName, shortTitle",
		ShortTitle:    "shortTitle",
		LongTitle:     "longTitle",
		Note:          "note",
	}
	contacts := []*directory.Contact{
		{ContactType: directory.ContactType_PHONE, Value: "+14155555555", Label: "Phone"},
		{ContactType: directory.ContactType_EMAIL, Value: "someone@example.com", Label: "Email"},
	}
	serializedContacts := []*directory.SerializedClientEntityContact{
		{
			EntityID:                entityID,
			Platform:                directory.Platform_IOS,
			SerializedEntityContact: []byte("{\"data\":\"serialized\"}")},
	}
	g.dirC.Expect(mock.NewExpectation(g.dirC.UpdateEntity, &directory.UpdateEntityRequest{
		EntityID:                 entityID,
		EntityInfo:               entityInfo,
		Contacts:                 contacts,
		SerializedEntityContacts: serializedContacts,
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
	}).WithReturns(&directory.UpdateEntityResponse{
		Entity: &directory.Entity{
			ID:       entityID,
			Contacts: contacts,
			Info:     entityInfo,
		},
	}, nil))
	g.dirC.Expect(mock.NewExpectation(g.dirC.SerializedEntityContact, &directory.SerializedEntityContactRequest{
		EntityID: entityID,
		Platform: directory.Platform_IOS,
	}).WithReturns(&directory.SerializedEntityContactResponse{
		SerializedEntityContact: &directory.SerializedClientEntityContact{
			EntityID:                entityID,
			Platform:                directory.Platform_IOS,
			SerializedEntityContact: []byte("{\"data\":\"serialized\"}"),
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ ($entityID: ID!) {
			updateEntity(input: {
				clientMutationId: "a1b2c3",
				entityID: $entityID,
				entityInfo: {
					firstName:     "firstName",
					middleInitial: "middleInitial",
					lastName:      "lastName",
					groupName:     "groupName",
					shortTitle:    "shortTitle",
					longTitle:     "longTitle",
					note:          "note",
					contactInfos:  [
						{type: PHONE, value: "+14155555555", label: "Phone"},
						{type: EMAIL, value: "someone@example.com", label: "Email"},
					],
					serializedContacts: [
						{platform: IOS, contact: "{\"data\":\"serialized\"}"},
					],
				},
			}) {
				clientMutationId
				entity {
					id
					firstName
					middleInitial
					lastName
					groupName
					displayName
					shortTitle
					longTitle
					note
					contacts {
						type
						value
						label
					}
					serializedContact(platform: IOS)
				}
			}
		}`, map[string]interface{}{
		"entityID": entityID,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"updateEntity": {
			"clientMutationId": "a1b2c3",
			"entity": {
				"contacts": [
					{
						"label": "Phone",
						"type": "PHONE",
						"value": "+14155555555"
					},
					{
						"label": "Email",
						"type": "EMAIL",
						"value": "someone@example.com"
					}
				],
				"displayName": "firstName middleInitial. lastName, shortTitle",
				"firstName": "firstName",
				"groupName": "groupName",
				"id": "e_1",
				"lastName": "lastName",
				"longTitle": "longTitle",
				"middleInitial": "middleInitial",
				"note": "note",
				"serializedContact": "{\"data\":\"serialized\"}",
				"shortTitle": "shortTitle"
			}
		}
	}
}`, string(b))
}
