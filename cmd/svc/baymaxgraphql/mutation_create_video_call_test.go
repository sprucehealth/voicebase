package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
)

func TestCreateVideoCallMutation(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.Entities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: "account_1",
			},
			RequestedInformation: &directory.RequestedInformation{
				EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_INTERNAL},
		}).WithReturns(
		[]*directory.Entity{{
			ID:        "entity_1",
			AccountID: "account_1",
			Type:      directory.EntityType_INTERNAL,
			Status:    directory.EntityStatus_ACTIVE,
			Memberships: []*directory.Entity{{
				ID:   "org",
				Type: directory.EntityType_ORGANIZATION,
			}},
		}}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: "entity_2",
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_PATIENT, directory.EntityType_INTERNAL},
		}).WithReturns(
		[]*directory.Entity{{
			ID:        "entity_2",
			AccountID: "account_2",
			Type:      directory.EntityType_PATIENT,
			Status:    directory.EntityStatus_ACTIVE,
		}}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.InitiateIPCall,
		&excomms.InitiateIPCallRequest{
			Type:               excomms.IPCallType_VIDEO,
			CallerEntityID:     "entity_1",
			RecipientEntityIDs: []string{"entity_2"},
		}).WithReturns(
		&excomms.InitiateIPCallResponse{
			Call: &excomms.IPCall{
				ID:    "ipc_123",
				Token: "token",
				Participants: []*excomms.IPCallParticipant{
					{
						EntityID:  "entity_1",
						AccountID: "account_1",
						Identity:  "identity_1",
						State:     excomms.IPCallState_ACCEPTED,
						Role:      excomms.IPCallParticipantRole_CALLER,
					},
					{
						EntityID:  "entity_2",
						AccountID: "account_2",
						Identity:  "identity_2",
						State:     excomms.IPCallState_PENDING,
						Role:      excomms.IPCallParticipantRole_RECIPIENT,
					},
				},
			},
		}, nil))

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account_1",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	res := g.query(ctx, `
		mutation _ {
			createVideoCall(input: {
				organizationID: "org",
				recipientCallEndpointIDs: ["entity_2"],
			}) {
				success
				call {
					id
					accessToken
					role
					caller {
						state
						twilioIdentity
					}
					recipients {
						state
						twilioIdentity
					}
					allowVideo
					videoEnabledByDefault
				}
			}
		}`, nil)
	responseEquals(t, `{
	"data": {
		"createVideoCall": {
			"call": {
				"accessToken": "token",
				"allowVideo": true,
				"caller": {
					"state": "ACCEPTED",
					"twilioIdentity": "identity_1"
				},
				"id": "ipc_123",
				"recipients": [{
					"state": "PENDING",
					"twilioIdentity": "identity_2"
				}],
				"role": "CALLER",
				"videoEnabledByDefault": true
			},
			"success": true
		}
	}
}`, res)
}
