package twilio

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/excomms"
)

// ParamsFromRequest parses the params from the request if present. An empty params struct is returned
// if no parameters are present. It is up to the caller to handle missing parameters.
func ParamsFromRequest(r *http.Request) (*excomms.TwilioParams, error) {
	t := &excomms.TwilioParams{
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
	}

	t.CallStatus = parseCallStatus(r.FormValue("CallStatus"))
	t.DialCallStatus = parseCallStatus(r.FormValue("DialCallStatus"))

	switch r.FormValue("Direction") {
	case "inbound":
		t.Direction = excomms.TwilioParams_INBOUND
	case "outbound-dial":
		t.Direction = excomms.TwilioParams_OUTBOUND_DIAL
	case "outbound-api":
		t.Direction = excomms.TwilioParams_OUTBOUND_API
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
		t.MediaItems = make([]*excomms.TwilioParams_TwilioMediaItem, t.NumMedia)
		for i := 0; i < int(t.NumMedia); i++ {
			t.MediaItems[i] = &excomms.TwilioParams_TwilioMediaItem{
				ContentType: r.FormValue(fmt.Sprintf("MediaContentType%d", i)),
				MediaURL:    r.FormValue(fmt.Sprintf("MediaUrl%d", i)),
			}
		}
	}

	return t, nil
}

func parseCallStatus(status string) excomms.TwilioParams_CallStatus {
	switch status {
	case "queued":
		return excomms.TwilioParams_QUEUED
	case "ringing":
		return excomms.TwilioParams_RINGING
	case "in-progress":
		return excomms.TwilioParams_IN_PROGRESS
	case "completed":
		return excomms.TwilioParams_COMPLETED
	case "busy":
		return excomms.TwilioParams_BUSY
	case "failed":
		return excomms.TwilioParams_FAILED
	case "no-answer":
		return excomms.TwilioParams_NO_ANSWER
	case "canceled":
		return excomms.TwilioParams_CANCELED
	case "answered":
		return excomms.TwilioParams_ANSWERED
	case "initiated":
		return excomms.TwilioParams_INITIATED
	}
	return excomms.TwilioParams_CALL_STATUS_UNDEFINED
}
