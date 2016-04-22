package twilio

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/cleaner"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/twilio/twiml"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
)

// STEP: Process outgoing call that is being made. Search to ensure that an active reservation
// for the outgoing call exists (identified by the source of the number and the proxy number
// being called)

func processOutgoingCall(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	originatingPhoneNumber, err := phone.ParseNumber(params.From)
	if err != nil {
		return "", errors.Trace(err)
	}

	proxyPhoneNumber, err := phone.ParseNumber(params.To)
	if err != nil {
		return "", errors.Trace(err)
	}

	// look for an active reservation on the proxy phone number
	ppnr, err := eh.proxyNumberManager.ActiveReservation(originatingPhoneNumber, proxyPhoneNumber)
	if err != nil {
		golog.Errorf(err.Error())
		return twiml.NewResponse(
			&twiml.Say{
				Text:  "Outbound calls to patients should be initiated from within the Spruce app. Please hang up and call the patient you are trying to reach by tapping the phone icon within their conversation thread. Thank you!",
				Voice: "alice",
			}).GenerateTwiML()
	}

	// look up the practice phone number using the organizationID
	res, err := eh.directory.LookupEntities(
		context.Background(),
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: ppnr.OrganizationID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 1,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERS,
					directory.EntityInformation_MEMBERSHIPS,
					directory.EntityInformation_CONTACTS,
				},
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		})
	if err != nil {
		return "", errors.Trace(err)
	} else if len(res.Entities) != 1 {
		return "", errors.Trace(fmt.Errorf("Expected 1 entity. Got %d", len(res.Entities)))
	}

	orgEntity := res.Entities[0]
	if orgEntity.Type != directory.EntityType_ORGANIZATION {
		return "", errors.Trace(fmt.Errorf("Expected entity to be of type %s but got type %s", directory.EntityType_ORGANIZATION.String(), orgEntity.Type.String()))
	}

	var practicePhoneNumber string
	for _, c := range orgEntity.Contacts {
		if c.Provisioned && c.ContactType == directory.ContactType_PHONE {
			practicePhoneNumber = c.Value
		}
	}
	if practicePhoneNumber == "" {
		return "", errors.Trace(fmt.Errorf("Unable to find practice phone number for org %s", orgEntity.ID))
	}

	if err := eh.proxyNumberManager.CallStarted(originatingPhoneNumber, proxyPhoneNumber); err != nil {
		return "", errors.Trace(err)
	}

	if err := eh.dal.CreateCallRequest(&models.CallRequest{
		Source:         originatingPhoneNumber,
		Destination:    ppnr.DestinationPhoneNumber,
		Proxy:          proxyPhoneNumber,
		OrganizationID: ppnr.OrganizationID,
		CallSID:        params.CallSID,
		Requested:      eh.clock.Now(),
		CallerEntityID: ppnr.OwnerEntityID,
		CalleeEntityID: ppnr.DestinationEntityID,
	}); err != nil {
		return "", errors.Trace(err)
	}

	// lookup external entity for name
	res, err = eh.directory.LookupEntities(
		context.Background(),
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: ppnr.DestinationEntityID,
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		})
	if err != nil {
		return "", errors.Trace(err)
	} else if len(res.Entities) != 1 {
		return "", errors.Trace(fmt.Errorf("Expected 1 entity. Got %d", len(res.Entities)))
	}

	var text string
	if res.Entities[0].Info != nil && res.Entities[0].Info.DisplayName != "" {
		text = "You will be connected to " + res.Entities[0].Info.DisplayName
	} else {
		formattedNumber, err := ppnr.DestinationPhoneNumber.Format(phone.National)
		if err != nil {
			golog.Errorf(err.Error())
			text = "You will be connected to the patient"
		} else {
			text = "You will be connected to " + formattedNumber
		}
	}

	tw := twiml.NewResponse(
		&twiml.Say{
			Text:  text,
			Voice: "alice",
		},
		&twiml.Dial{
			CallerID: practicePhoneNumber,
			Nouns: []interface{}{
				&twiml.Number{
					StatusCallbackEvent: twiml.SCRinging | twiml.SCAnswered | twiml.SCCompleted,
					StatusCallback:      fmt.Sprintf("%s/twilio/call/process_outgoing_call_status", eh.apiURL),
					Text:                ppnr.DestinationPhoneNumber.String(),
				},
			},
		})

	return tw.GenerateTwiML()
}

// STEP: Process status of outgoing call and explicitly handle certain dial out use-cases
// to insert those as events into the thread.

func processOutgoingCallStatus(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	// NOTE: We use the callSID of the parent call to process the status of the outgoing
	// call placed as the outgoing call is dialed out via a separate call leg.
	// This is under the assumption that the outgoing call from provider to external
	// entity was placed via the Dial verb.
	if params.ParentCallSID == "" {
		golog.Debugf("Nothing to do because params.ParentCallSID is empty")
		// nothing to do
		return "", nil
	}

	cr, err := eh.dal.LookupCallRequest(params.ParentCallSID)
	if errors.Cause(err) == dal.ErrCallRequestNotFound {
		return "", errors.Trace(fmt.Errorf("No call requeht found for call sid %s", params.ParentCallSID))
	} else if err != nil {
		return "", errors.Trace(err)
	}

	var cet *excomms.PublishedExternalMessage_Outgoing
	switch params.CallStatus {
	case rawmsg.TwilioParams_RINGING:
		cet = &excomms.PublishedExternalMessage_Outgoing{
			Outgoing: &excomms.OutgoingCallEventItem{
				Type:              excomms.OutgoingCallEventItem_PLACED,
				DurationInSeconds: params.CallDuration,
				CallerEntityID:    cr.CallerEntityID,
				CalleeEntityID:    cr.CalleeEntityID,
			},
		}
	case rawmsg.TwilioParams_ANSWERED, rawmsg.TwilioParams_COMPLETED:
		cet = &excomms.PublishedExternalMessage_Outgoing{
			Outgoing: &excomms.OutgoingCallEventItem{
				Type:              excomms.OutgoingCallEventItem_ANSWERED,
				DurationInSeconds: params.CallDuration,
				CallerEntityID:    cr.CallerEntityID,
				CalleeEntityID:    cr.CalleeEntityID,
			},
		}
		if err := eh.proxyNumberManager.CallEnded(cr.Source, cr.Proxy); err != nil {
			return "", errors.Trace(err)
		}

		cleaner.Publish(eh.sns, eh.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_CALL,
			ResourceID: params.CallSID,
		})

		cleaner.Publish(eh.sns, eh.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_CALL,
			ResourceID: params.ParentCallSID,
		})

		trackOutboundCall(eh, cr.CallerEntityID, cr.OrganizationID, cr.Destination.String(), params.CallDuration)
	default:
		// nothing to do
		golog.Debugf("Ignoring call status %s", params.CallStatus.String())
		return "", nil
	}

	conc.Go(func() {
		if err := sns.Publish(eh.sns, eh.externalMessageTopic, &excomms.PublishedExternalMessage{
			FromChannelID: cr.Source.String(),
			ToChannelID:   cr.Destination.String(),
			Direction:     excomms.PublishedExternalMessage_OUTBOUND,
			Timestamp:     uint64(time.Now().Unix()),
			Type:          excomms.PublishedExternalMessage_OUTGOING_CALL_EVENT,
			Item:          cet,
		}); err != nil {
			golog.Errorf(err.Error())
		}
	})

	return "", nil
}

// STEP: Process status of outgoing call

func processOutgoingSMSStatus(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	switch params.MessageStatus {
	case rawmsg.TwilioParams_MSG_STATUS_DELIVERED:
		cleaner.Publish(eh.sns, eh.resourceCleanerTopic, &models.DeleteResourceRequest{
			Type:       models.DeleteResourceRequest_TWILIO_SMS,
			ResourceID: params.MessageSID,
		})
	case rawmsg.TwilioParams_MSG_STATUS_FAILED:
		// for now if message sending failed lets log error so that we
		// can investigate why a message that passed validation failed to send
		golog.Errorf("Failed to send message %s", params.MessageSID)
	}
	return "", nil
}
