package twilio

import (
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestCheckResponse(t *testing.T) {
	res := &http.Response{
		Request:    &http.Request{},
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(strings.NewReader(`{"status": 400, "code": 21201, "message": "invalid parameter"}`)),
	}
	err, ok := CheckResponse(res).(*Exception)
	if !ok || err == nil {
		t.Error("CheckResponse expected error response")
	}

	want := &Exception{
		Status:  400,
		Code:    21201,
		Message: "invalid parameter",
	}

	if !reflect.DeepEqual(err, want) {
		t.Errorf("Exception = %#v, want %#v", err, want)
	}

	// XML

	res = &http.Response{
		Request:    &http.Request{},
		StatusCode: http.StatusBadRequest,
		Body: ioutil.NopCloser(strings.NewReader(`
			<?xml version='1.0' encoding='UTF-8'?>
			<TwilioResponse>
				<RestException>
					<Code>20404</Code>
					<Message>The requested resource /2010-04-01/Accounts/111/Messages/222/Media/333 was not found</Message>
					<MoreInfo>https://www.twilio.com/docs/errors/20404</MoreInfo>
					<Status>404</Status>
				</RestException>
			</TwilioResponse>`)),
	}
	err, ok = CheckResponse(res).(*Exception)
	if !ok || err == nil {
		t.Error("CheckResponse expected error response")
	}

	want = &Exception{
		Status:   404,
		Code:     20404,
		Message:  "The requested resource /2010-04-01/Accounts/111/Messages/222/Media/333 was not found",
		MoreInfo: "https://www.twilio.com/docs/errors/20404",
	}

	if !reflect.DeepEqual(err, want) {
		t.Errorf("Exception = %#v, want %#v", err, want)
	}
}
