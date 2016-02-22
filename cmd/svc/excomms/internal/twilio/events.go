package twilio

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/twilio/twiml"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
)

var (
	twilioEventsHandlers = map[rawmsg.TwilioEvent]twilioEventHandleFunc{
		rawmsg.TwilioEvent_PROCESS_INCOMING_CALL:        processIncomingCall,
		rawmsg.TwilioEvent_PROCESS_OUTGOING_CALL:        processOutgoingCall,
		rawmsg.TwilioEvent_PROVIDER_ENTERED_DIGITS:      providerEnteredDigits,
		rawmsg.TwilioEvent_PROVIDER_CALL_CONNECTED:      providerCallConnected,
		rawmsg.TwilioEvent_TWIML_REQUESTED_VOICEMAIL:    voicemailTWIML,
		rawmsg.TwilioEvent_PROCESS_INCOMING_CALL_STATUS: processIncomingCallStatus,
		rawmsg.TwilioEvent_PROCESS_VOICEMAIL:            processVoicemail,
		rawmsg.TwilioEvent_PROCESS_OUTGOING_CALL_STATUS: processOutgoingCallStatus,
	}
	maxPhoneNumbers = 10
)

type eventsHandler struct {
	directory            directory.DirectoryClient
	settings             settings.SettingsClient
	dal                  dal.DAL
	sns                  snsiface.SNSAPI
	clock                clock.Clock
	proxyNumberManager   proxynumber.Manager
	apiURL               string
	externalMessageTopic string
	incomingRawMsgTopic  string
}

func NewEventHandler(directory directory.DirectoryClient, settingsClient settings.SettingsClient, dal dal.DAL, sns snsiface.SNSAPI, clock clock.Clock, proxyNumberManager proxynumber.Manager, apiURL, externalMessageTopic, incomingRawMsgTopic string) EventHandler {
	return &eventsHandler{
		directory:            directory,
		settings:             settingsClient,
		dal:                  dal,
		clock:                clock,
		sns:                  sns,
		apiURL:               apiURL,
		externalMessageTopic: externalMessageTopic,
		incomingRawMsgTopic:  incomingRawMsgTopic,
		proxyNumberManager:   proxyNumberManager,
	}
}

func (e *eventsHandler) Process(ctx context.Context, event rawmsg.TwilioEvent, params *rawmsg.TwilioParams) (string, error) {
	handler := twilioEventsHandlers[event]
	if handler == nil {
		return "", fmt.Errorf("unknown event: %s", event.String())
	}
	twiml, err := handler(ctx, params, e)
	if err != nil {
		return "", errors.Trace(err)
	}

	conc.Go(func() {
		if err := e.dal.LogCallEvent(&models.CallEvent{
			Data:        params,
			Type:        event.String(),
			Source:      params.From,
			Destination: params.To,
		}); err != nil {
			golog.Errorf("Unable to log event %s: %s", event.String(), err.Error())
		}
	})
	return twiml, nil
}

type EventHandler interface {
	Process(ctx context.Context, event rawmsg.TwilioEvent, params *rawmsg.TwilioParams) (string, error)
}

type twilioEventHandleFunc func(context.Context, *rawmsg.TwilioParams, *eventsHandler) (string, error)

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
		return "", errors.Trace(err)
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

func processIncomingCall(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	golog.Infof("Incoming call %s to %s.", params.From, params.To)

	// lookup the entity for the destination of the incoming call
	res, err := eh.directory.LookupEntitiesByContact(
		context.Background(),
		&directory.LookupEntitiesByContactRequest{
			ContactValue: params.To,
			RequestedInformation: &directory.RequestedInformation{
				Depth: 2,
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
	}

	// we should get back a single entity at this point given that there should be a 1:1 mapping between a provisioned number
	// and an entity
	if len(res.Entities) != 1 {
		return "", errors.Trace(fmt.Errorf("Expected 1 entity for provisioned number, got back %d", len(res.Entities)))
	}

	var forwardingList []string
	var providersInForwardingList map[string]bool
	var phoneNumberToProviderMap map[string]string
	var organizationID string
	switch res.Entities[0].Type {
	case directory.EntityType_ORGANIZATION:
		organizationID = res.Entities[0].ID

		forwardingList, err = getForwardingListForProvisionedPhoneNumber(ctx, params.To, organizationID, eh)
		if err != nil {
			return "", errors.Trace(err)
		}

		// track the phone numbers in the forwarding list that map to a provider
		// we then need to check if any of the providers want their calls directed to voicemail.
		providersInForwardingList = make(map[string]bool, len(forwardingList))
		phoneNumberToProviderMap = make(map[string]string, len(forwardingList))
		for _, pn := range forwardingList {
			parsedPn, err := phone.Format(pn, phone.E164)
			if err != nil {
				golog.Errorf("Unable to parse phone number %s: %s", pn, err.Error())
				continue
			}

			for _, m := range res.Entities[0].Members {
				for _, c := range m.Contacts {
					if c.Value == parsedPn {
						providersInForwardingList[m.ID] = true
						phoneNumberToProviderMap[parsedPn] = m.ID
					}
				}
			}
		}
	case directory.EntityType_INTERNAL:
		for _, c := range res.Entities[0].Contacts {
			if c.Provisioned {
				continue
			} else if c.ContactType != directory.ContactType_PHONE {
				continue
			}
			// assuming for now that we are to call the first non-provisioned
			// phone number mapped to the provider.
			forwardingList = []string{c.Value}
			providersInForwardingList = map[string]bool{res.Entities[0].ID: true}
			phoneNumberToProviderMap = map[string]string{c.Value: res.Entities[0].ID}
			break
		}

		for _, m := range res.Entities[0].Memberships {
			if m.Type == directory.EntityType_ORGANIZATION {
				organizationID = m.ID
				break
			}
		}
	default:
		return "", errors.Trace(fmt.Errorf("Unexpected entity type %s", res.Entities[0].Type.String()))
	}

	if organizationID == "" {
		return "", errors.Trace(fmt.Errorf("Unable to find organization for provisioned number %s", params.To))
	}

	// remove the providers from the forwarding list that have a setting
	// turned on to indicate that all calls should be directed to voicemail
	par := conc.NewParallel()
	sendAllCallsToVoicemailMap := conc.NewMap()
	for entityID := range providersInForwardingList {
		eID := entityID
		par.Go(func() error {
			val, err := sendAllCallsToVoicemailForProvider(ctx, eID, eh)
			if err != nil {
				return err
			}
			sendAllCallsToVoicemailMap.Set(eID, val)
			return nil
		})
	}
	if err := par.Wait(); err != nil {
		return "", errors.Trace(err)
	}

	numbers := make([]interface{}, 0, maxPhoneNumbers)
	for _, p := range forwardingList {
		if len(numbers) == maxPhoneNumbers {
			golog.Errorf("Org %s is currently configured to simultaneously call more than 10 numbers when that is the maximum that twilio supports.", organizationID)
			break
		}

		// check if send all calls to voicemail setting is on
		// for any provider in the forwarding list
		eID, ok := phoneNumberToProviderMap[p]
		if ok {
			val := sendAllCallsToVoicemailMap.Get(eID)
			if val != nil && val.(bool) {
				// skip including number from the list if provider indicated
				// that they want all calls to be sent to voicemail
				continue
			}
		}

		numbers = append(numbers, &twiml.Number{
			URL:  "/twilio/call/provider_call_connected",
			Text: p,
		})
	}

	// if there are no numbers in the forwarding list, then direct calls to voicemail
	if len(numbers) == 0 {
		return voicemailTWIML(ctx, params, eh)
	}

	tw := twiml.NewResponse(
		&twiml.Dial{
			CallerID:         params.To,
			TimeoutInSeconds: 30,
			Action:           "/twilio/call/process_incoming_call_status",
			Nouns:            numbers,
		},
	)

	return tw.GenerateTwiML()
}

func providerCallConnected(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	golog.Infof("Call connected for provider at %s.", params.To)

	tw := twiml.NewResponse(
		&twiml.Gather{
			Action:           "/twilio/call/provider_entered_digits",
			Method:           "POST",
			TimeoutInSeconds: 5,
			NumDigits:        1,
			Verbs: []interface{}{
				&twiml.Say{
					Voice: "woman",
					Text:  "You have an incoming call. Press 1 to answer.",
				},
			},
		},
		// In the event that no key is entered, we hang up the
		// dialed call to then direct the caller to voicemail.
		&twiml.Hangup{},
	)

	return tw.GenerateTwiML()
}

func providerEnteredDigits(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	golog.Infof("Provider entered digits %s at %s.", params.Digits, params.To)

	if params.Digits == "1" {
		// accept the call if the provider entered the right digit
		// by generating an empty response.
		tw := twiml.NewResponse()
		return tw.GenerateTwiML()
	}

	// hangup they key on the provider side if any key other than 1 is pressed.
	tw := twiml.NewResponse(&twiml.Hangup{})
	return tw.GenerateTwiML()
}

func voicemailTWIML(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	// TODO: Configurable voice mail or default mehsage based on user configuration.
	tw := &twiml.Response{
		Verbs: []interface{}{
			&twiml.Play{
				Text: "http://dev-twilio.s3.amazonaws.com/kunal_clinic_voicemail.mp3",
			},
			&twiml.Record{
				Action:           "/twilio/call/process_voicemail",
				PlayBeep:         true,
				TimeoutInSeconds: 60,
			},
		},
	}

	return tw.GenerateTwiML()
}

func processIncomingCallStatus(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {
	switch params.DialCallStatus {
	case rawmsg.TwilioParams_ANSWERED, rawmsg.TwilioParams_COMPLETED:
		conc.Go(func() {
			if err := sns.Publish(eh.sns, eh.externalMessageTopic, &excomms.PublishedExternalMessage{
				FromChannelID: params.From,
				ToChannelID:   params.To,
				Timestamp:     uint64(time.Now().Unix()),
				Direction:     excomms.PublishedExternalMessage_INBOUND,
				Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
				Item: &excomms.PublishedExternalMessage_Incoming{
					Incoming: &excomms.IncomingCallEventItem{
						Type:              excomms.IncomingCallEventItem_ANSWERED,
						DurationInSeconds: params.CallDuration,
					},
				},
			}); err != nil {
				golog.Errorf(err.Error())
			}
		})

	case rawmsg.TwilioParams_CALL_STATUS_UNDEFINED:
	default:
		return voicemailTWIML(ctx, params, eh)
	}

	return "", nil
}

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

func processVoicemail(ctx context.Context, params *rawmsg.TwilioParams, eh *eventsHandler) (string, error) {

	rawMessageID, err := eh.dal.StoreIncomingRawMessage(&rawmsg.Incoming{
		Type: rawmsg.Incoming_TWILIO_VOICEMAIL,
		Message: &rawmsg.Incoming_Twilio{
			Twilio: params,
		},
	})
	if err != nil {
		return "", errors.Trace(err)
	}

	conc.Go(func() {
		if err := sns.Publish(eh.sns, eh.incomingRawMsgTopic, &sns.IncomingRawMessageNotification{
			ID: rawMessageID,
		}); err != nil {
			golog.Errorf(err.Error())
		}
	})

	return "", nil
}

func getForwardingListForProvisionedPhoneNumber(ctx context.Context, phoneNumber, organizationID string, eh *eventsHandler) ([]string, error) {

	settingsRes, err := eh.settings.GetValues(ctx, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyForwardingList,
				Subkey: phoneNumber,
			},
		},
		NodeID: organizationID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(settingsRes.Values) != 1 {
		return nil, errors.Trace(fmt.Errorf("Expected single value for forwarding list of provisioned phone number %s but got back %d", phoneNumber, len(settingsRes.Values)))
	} else if settingsRes.Values[0].GetStringList() == nil {
		return nil, errors.Trace(fmt.Errorf("Expected string list value but got %T", settingsRes.Values[0]))
	}

	forwardingListMap := make(map[string]bool, len(settingsRes.Values[0].GetStringList().Values))
	forwardingList := make([]string, 0, len(settingsRes.Values[0].GetStringList().Values))
	for _, s := range settingsRes.Values[0].GetStringList().Values {
		if forwardingListMap[s] {
			continue
		}
		forwardingListMap[s] = true
		forwardingList = append(forwardingList, s)
	}

	return forwardingList, nil
}

func sendAllCallsToVoicemailForProvider(ctx context.Context, entityID string, eh *eventsHandler) (bool, error) {
	res, err := eh.settings.GetValues(ctx, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key: excommsSettings.ConfigKeySendCallsToVoicemail,
			},
		},
		NodeID: entityID,
	})
	if err != nil {
		return false, errors.Trace(err)
	} else if len(res.Values) != 1 {
		return false, errors.Trace(fmt.Errorf("Expected 1 value for config %s but got %d", excommsSettings.ConfigKeySendCallsToVoicemail, len(res.Values)))
	} else if res.Values[0].GetBoolean() == nil {
		return false, errors.Trace(fmt.Errorf("Expected boolean value for config %s but got %T", excommsSettings.ConfigKeySendCallsToVoicemail, res.Values[0]))
	}

	return res.Values[0].GetBoolean().Value, nil
}
