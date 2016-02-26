package twilio

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/libs/errors"
)

// ParamsFromRequest parses the params from the request if present. An empty params struct is returned
// if no parameters are present. It is up to the caller to handle missing parameters.
func ParamsFromRequest(r *http.Request) (*rawmsg.TwilioParams, error) {
	t := &rawmsg.TwilioParams{
		CallSID:            r.FormValue("CallSid"),
		AccountSID:         r.FormValue("AccountSid"),
		From:               r.FormValue("From"),
		To:                 r.FormValue("To"),
		APIVersion:         r.FormValue("ApiVersion"),
		ForwardedFrom:      r.FormValue("ForwardedFrom"),
		CallerName:         r.FormValue("CallerName"),
		FromCity:           r.FormValue("FromCity"),
		FromState:          r.FormValue("FromState"),
		FromZip:            r.FormValue("FromZip"),
		FromCountry:        r.FormValue("FromCountry"),
		ToCity:             r.FormValue("ToCity"),
		ToState:            r.FormValue("ToState"),
		ToZip:              r.FormValue("ToZip"),
		ToCountry:          r.FormValue("ToCountry"),
		RecordingURL:       r.FormValue("RecordingUrl"),
		RecordingSID:       r.FormValue("RecordingSid"),
		Digits:             r.FormValue("Digits"),
		MessageSID:         r.FormValue("MessageSid"),
		SMSSID:             r.FormValue("SmsSid"),
		MessagingServiceID: r.FormValue("MessagingServiceId"),
		Body:               r.FormValue("Body"),
		QueueSID:           r.FormValue("QueueSid"),
		DequeingCallSID:    r.FormValue("DequeingCallSid"),
		ParentCallSID:      r.FormValue("ParentCallSid"),
		DialCallSID:        r.FormValue("DialCallSid"),
	}

	t.CallStatus = parseCallStatus(r.FormValue("CallStatus"))
	t.MessageStatus = parseMessageStatus(r.FormValue("MessageStatus"))
	t.DialCallStatus = parseCallStatus(r.FormValue("DialCallStatus"))

	switch r.FormValue("Direction") {
	case "inbound":
		t.Direction = rawmsg.TwilioParams_INBOUND
	case "outbound-dial":
		t.Direction = rawmsg.TwilioParams_OUTBOUND_DIAL
	case "outbound-api":
		t.Direction = rawmsg.TwilioParams_OUTBOUND_API
	}

	if rd := r.FormValue("RecordingDuration"); rd != "" {
		recordingDuration, err := strconv.Atoi(rd)
		if err != nil {
			return nil, errors.Trace(err)
		}
		t.RecordingDuration = uint32(recordingDuration)
	}

	if qt := r.FormValue("QueueTime"); qt != "" {
		queueTime, err := strconv.Atoi(qt)
		if err != nil {
			return nil, errors.Trace(err)
		}
		t.QueueTime = uint32(queueTime)
	}

	if cd := r.FormValue("CallDuration"); cd != "" {
		callDuration, err := strconv.Atoi(cd)
		if err != nil {
			return nil, errors.Trace(err)
		}
		t.CallDuration = uint32(callDuration)
	}

	if cd := r.FormValue("DialCallDuration"); cd != "" {
		callDuration, err := strconv.Atoi(cd)
		if err != nil {
			return nil, errors.Trace(err)
		}
		t.DialCallDuration = uint32(callDuration)
	}

	if nm := r.FormValue("NumMedia"); nm != "" {
		numMedia, err := strconv.Atoi(nm)
		if err != nil {
			return nil, errors.Trace(err)
		}
		t.NumMedia = uint32(numMedia)
	}

	if t.NumMedia > 0 {
		t.MediaItems = make([]*rawmsg.TwilioParams_TwilioMediaItem, t.NumMedia)
		for i := 0; i < int(t.NumMedia); i++ {
			t.MediaItems[i] = &rawmsg.TwilioParams_TwilioMediaItem{
				ContentType: r.FormValue(fmt.Sprintf("MediaContentType%d", i)),
				MediaURL:    r.FormValue(fmt.Sprintf("MediaUrl%d", i)),
			}
		}
	}

	return t, nil
}

func parseCallStatus(status string) rawmsg.TwilioParams_CallStatus {
	switch status {
	case "queued":
		return rawmsg.TwilioParams_QUEUED
	case "ringing":
		return rawmsg.TwilioParams_RINGING
	case "in-progress":
		return rawmsg.TwilioParams_IN_PROGRESS
	case "completed":
		return rawmsg.TwilioParams_COMPLETED
	case "busy":
		return rawmsg.TwilioParams_BUSY
	case "failed":
		return rawmsg.TwilioParams_FAILED
	case "no-answer":
		return rawmsg.TwilioParams_NO_ANSWER
	case "canceled":
		return rawmsg.TwilioParams_CANCELED
	case "answered":
		return rawmsg.TwilioParams_ANSWERED
	case "initiated":
		return rawmsg.TwilioParams_INITIATED
	}
	return rawmsg.TwilioParams_CALL_STATUS_UNDEFINED
}

func parseMessageStatus(status string) rawmsg.TwilioParams_MessageStatus {
	switch status {
	case "queued":
		return rawmsg.TwilioParams_MSG_STATUS_QUEUED
	case "sending":
		return rawmsg.TwilioParams_MSG_STATUS_SENDING
	case "sent":
		return rawmsg.TwilioParams_MSG_STATUS_SENT
	case "failed":
		return rawmsg.TwilioParams_MSG_STATUS_FAILED
	case "delivered":
		return rawmsg.TwilioParams_MSG_STATUS_DELIVERED
	case "undelivered":
		return rawmsg.TwilioParams_MSG_STATUS_UNDELIVERED
	}
	return rawmsg.TwilioParams_MSG_STATUS_INVALID
}
