package twilio

import (
	"fmt"
	"strings"

	"context"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
)

func trackInboundCall(eh *eventsHandler, callSID, eventSuffix string) {
	conc.Go(func() {
		incomingCall, err := eh.dal.LookupIncomingCall(callSID)
		if err != nil {
			golog.Errorf("Unable to lookup incoming call %s: %s", callSID, err.Error())
			return
		}

		res, err := eh.directory.LookupEntities(
			context.Background(),
			&directory.LookupEntitiesRequest{
				LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
				LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
					EntityID: incomingCall.OrganizationID,
				},
				RequestedInformation: &directory.RequestedInformation{
					Depth: 1,
					EntityInformation: []directory.EntityInformation{
						directory.EntityInformation_EXTERNAL_IDS,
						directory.EntityInformation_MEMBERS,
					},
				},
				Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			})
		if err != nil {
			golog.Errorf("Unable to lookup entity %s:%s", incomingCall.OrganizationID, err.Error())
			return
		} else if len(res.Entities) != 1 {
			golog.Errorf("Expected 1 entity but got %d for %s", len(res.Entities), err.Error())
			return
		}

		// because there is no easy way for us to know which member
		// of the org answered the call, for now associate the inbound call
		// with every member of the org.
		// the reason that it is hard for identify who answered the call is beacuse
		// the call list is a generic list of numbers that we'd have to check which
		// entity provider they map to, to see who actually answered the call.
		for _, member := range res.Entities[0].Members {

			if member.Type != directory.EntityType_INTERNAL {
				continue
			}

			accountID := determineAccountID(member)
			if accountID == "" {
				golog.Errorf("No accountID found for entity %s", member.ID)
				return
			}

			analytics.SegmentTrack(&segment.Track{
				Event:  fmt.Sprintf("inbound-call-%s", eventSuffix),
				UserId: accountID,
				Properties: map[string]interface{}{
					"destination": incomingCall.Destination,
				},
			})
		}
	})
}

func trackOutboundCall(eh *eventsHandler, callerEntityID, orgID, destination string, durationInSeconds uint32) {
	conc.Go(func() {
		res, err := eh.directory.LookupEntities(
			context.Background(),
			&directory.LookupEntitiesRequest{
				LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
				LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
					EntityID: callerEntityID,
				},
				RequestedInformation: &directory.RequestedInformation{
					Depth: 0,
					EntityInformation: []directory.EntityInformation{
						directory.EntityInformation_EXTERNAL_IDS,
					},
				},
				Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			})
		if err != nil {
			golog.Errorf("Unable to lookup entity %s: %s", callerEntityID, err)
		} else if len(res.Entities) != 1 {
			golog.Errorf("Expected 1 entity but got %d for %s", len(res.Entities), callerEntityID)
		}

		accountID := determineAccountID(res.Entities[0])
		if accountID == "" {
			golog.Errorf("No accountID found for entity %s", res.Entities[0].ID)
			return
		}

		analytics.SegmentTrack(&segment.Track{
			Event:  "outbound-call-connected",
			UserId: accountID,
			Properties: map[string]interface{}{
				"caller_entity_id":    callerEntityID,
				"org_id":              orgID,
				"duration_in_seconds": durationInSeconds,
			},
		})

	})
}

func determineAccountID(entity *directory.Entity) string {
	for _, externalID := range entity.ExternalIDs {
		if strings.HasPrefix(externalID, auth.AccountIDPrefix) {
			return externalID
		}
	}
	return ""
}
