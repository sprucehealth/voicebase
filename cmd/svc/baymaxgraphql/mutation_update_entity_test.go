package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestUpdateEntityMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "a_1",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	entityID := "e_1"

	entityInfo := &directory.EntityInfo{
		FirstName:     "firstName",
		MiddleInitial: "middleInitial",
		LastName:      "lastName",
		GroupName:     "groupName",
		ShortTitle:    "shortTitle",
		LongTitle:     "longTitle",
		Note:          "note",
	}

	returnedEntityInfo := &directory.EntityInfo{
		FirstName:     entityInfo.FirstName,
		MiddleInitial: entityInfo.MiddleInitial,
		LastName:      entityInfo.LastName,
		GroupName:     entityInfo.GroupName,
		ShortTitle:    entityInfo.ShortTitle,
		LongTitle:     entityInfo.LongTitle,
		Note:          entityInfo.Note,
		DisplayName:   "firstName middleInitial. lastName, shortTitle",
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
	g.ra.Expect(mock.NewExpectation(g.ra.UpdateEntity, &directory.UpdateEntityRequest{
		EntityID:                       entityID,
		UpdateEntityInfo:               true,
		EntityInfo:                     entityInfo,
		UpdateContacts:                 true,
		Contacts:                       contacts,
		UpdateSerializedEntityContacts: true,
		SerializedEntityContacts:       serializedContacts,
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
	}).WithReturns(&directory.Entity{
		ID:       entityID,
		Contacts: contacts,
		Info:     returnedEntityInfo,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.ThreadsForMember, entityID, true).WithReturns([]*threading.Thread{
		{
			ID: "t1",
		},
		{
			ID: "t2",
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.UpdateThread, &threading.UpdateThreadRequest{
		ThreadID:    "t1",
		SystemTitle: returnedEntityInfo.DisplayName,
	}))
	g.ra.Expect(mock.NewExpectation(g.ra.UpdateThread, &threading.UpdateThreadRequest{
		ThreadID:    "t2",
		SystemTitle: returnedEntityInfo.DisplayName,
	}))

	g.ra.Expect(mock.NewExpectation(g.ra.SerializedEntityContact, entityID, directory.Platform_IOS).WithReturns(
		&directory.SerializedClientEntityContact{
			EntityID:                entityID,
			Platform:                directory.Platform_IOS,
			SerializedEntityContact: []byte("{\"data\":\"serialized\"}"),
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
