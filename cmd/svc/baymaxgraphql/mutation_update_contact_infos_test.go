package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestUpdateContactInfosMutation(t *testing.T) {
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
		DisplayName:   "displayName",
		ShortTitle:    "shortTitle",
		LongTitle:     "longTitle",
		Note:          "note",
	}
	contacts := []*directory.Contact{
		{ContactType: directory.ContactType_PHONE, Value: "+14155555555", Label: "Phone"},
		{ContactType: directory.ContactType_EMAIL, Value: "someone@example.com", Label: "Email"},
	}
	g.ra.Expect(mock.NewExpectation(g.ra.UpdateContacts, &directory.UpdateContactsRequest{
		EntityID: entityID,
		Contacts: contacts,
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
	}).WithReturns(&directory.Entity{
		ID:       entityID,
		Contacts: contacts,
		Info:     entityInfo,
	}, nil))

	res := g.query(ctx, `
		mutation _ ($entityID: ID!) {
			updateContactInfos(input: {
				clientMutationId: "a1b2c3",
				entityID: $entityID,
				contactInfos:  [
					{type: PHONE, value: "+14155555555", label: "Phone"},
					{type: EMAIL, value: "someone@example.com", label: "Email"},
				],
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
				}
			}
		}`, map[string]interface{}{
		"entityID": entityID,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"updateContactInfos": {
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
				"displayName": "displayName",
				"firstName": "firstName",
				"groupName": "groupName",
				"id": "e_1",
				"lastName": "lastName",
				"longTitle": "longTitle",
				"middleInitial": "middleInitial",
				"note": "note",
				"shortTitle": "shortTitle"
			}
		}
	}
}`, string(b))
}
