package twiml

import "testing"

func TestDialResponse_Number(t *testing.T) {
	tw := Response{
		Verbs: []interface{}{
			&Dial{
				CallerID: "+11234567890",
				Nouns: []interface{}{
					&Number{
						StatusCallbackEvent: SCRinging | SCAnswered | SCCompleted,
						StatusCallback:      "http://www.google.com",
						Text:                "+17348465522",
					},
				},
			},
		},
	}

	str, err := tw.GenerateTwiML()
	if err != nil {
		t.Fatal(err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Dial callerId="+11234567890"><Number statusCallbackEvent="ringing answered completed" statusCallback="http://www.google.com">+17348465522</Number></Dial></Response>`

	if str != expected {
		t.Fatalf("Expected %s\nGot: %s", expected, str)
	}
}

func TestDialResponse_Text(t *testing.T) {
	tw := Response{
		Verbs: []interface{}{
			&Dial{
				CallerID:  "+11234567890",
				PlainText: "+17348465522",
			},
		},
	}

	str, err := tw.GenerateTwiML()
	if err != nil {
		t.Fatal(err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Dial callerId="+11234567890">+17348465522</Dial></Response>`

	if str != expected {
		t.Fatalf("Expected %s\nGot: %s", expected, str)
	}
}

func TestDialResponse_Client(t *testing.T) {
	tw := Response{
		Verbs: []interface{}{
			&Dial{
				Nouns: []interface{}{
					&Client{
						URL:    "http://www.google.com",
						Method: "POST",
						Text:   "jimmy",
					},
				},
			},
		},
	}

	str, err := tw.GenerateTwiML()
	if err != nil {
		t.Fatal(err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Dial><Client url="http://www.google.com" method="POST">jimmy</Client></Dial></Response>`

	if str != expected {
		t.Fatalf("Expected %s\nGot: %s", expected, str)
	}
}

func TestDialResponse_Queue(t *testing.T) {
	tw := Response{
		Verbs: []interface{}{
			&Dial{
				Nouns: []interface{}{
					&Queue{
						URL:  "http://www.google.com",
						Text: "support",
					},
				},
			},
		},
	}

	str, err := tw.GenerateTwiML()
	if err != nil {
		t.Fatal(err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Dial><Queue url="http://www.google.com">support</Queue></Dial></Response>`

	if str != expected {
		t.Fatalf("Expected %s\nGot: %s", expected, str)
	}
}

func TestEnqueueResponse_EnQueue(t *testing.T) {
	tw := Response{
		Verbs: []interface{}{
			&Enqueue{
				WaitURL: "/twilio/twiml_wait_music",
				Text:    "support",
			},
		},
	}

	str, err := tw.GenerateTwiML()
	if err != nil {
		t.Fatal(err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Enqueue waitUrl="/twilio/twiml_wait_music">support</Enqueue></Response>`

	if str != expected {
		t.Fatalf("Expected %s\nGot: %s", expected, str)
	}
}
