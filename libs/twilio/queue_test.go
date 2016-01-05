package twilio

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestQueueService_Get(t *testing.T) {
	setup()
	defer teardown()

	queueSID := "QUxxxxxxxxx"
	u := client.EndPoint("Queues", queueSID)

	output := `
	{
    "sid": "QUxxxxxxxxx",
    "friendly_name": "persistent_queue1",
    "current_size": 0,
    "average_wait_time": 0,
    "max_size": 10,
    "date_created": "Mon, 26 Mar 2012 22:00:14 +0000",
    "date_updated": "Mon, 26 Mar 2012 22:00:14 +0000",
    "uri": "/2010-04-01/Accounts/AC5ef87.../Queues/QUxxxxxxxxx.json"
	}	
	`

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprintf(w, output)
	})

	queue, _, err := client.Queue.Get(queueSID)
	if err != nil {
		t.Fatal(err)
	}

	tm := parseTimestamp("Mon, 26 Mar 2012 22:00:14 +0000")
	want := &Queue{
		SID:             queueSID,
		FriendlyName:    "persistent_queue1",
		CurrentSize:     0,
		AverageWaitTime: 0,
		MaxSize:         10,
		DateCreated:     tm,
		DateUpdated:     tm,
		URI:             "/2010-04-01/Accounts/AC5ef87.../Queues/QUxxxxxxxxx.json",
	}

	if !reflect.DeepEqual(queue, want) {
		t.Errorf("Queues.Get() returned %+v, want %+v", queue, want)
	}
}

func TestQueueService_Front(t *testing.T) {
	setup()
	defer teardown()

	queueSID := "QUxxxxxxxxx"
	u := client.EndPoint("Queues", queueSID, "Members", "Front")

	output :=
		`
	{
    "call_sid": "CA386025c9bf5d6052a1d1ea42b4d16662",
    "date_enqueued": "Mon, 4 Feb 2012 15:44:15 +0000",
    "wait_time": 30,
    "position": 1,
    "uri": "/2010-04-01/Accounts/AC5ef87.../Queues/QUxxxxxxxxx/Members/CA386025c9bf5d6052a1d1ea42b4d16662.json"
	}
	`

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprintf(w, output)
	})

	queueMember, _, err := client.Queue.Front(queueSID)
	if err != nil {
		t.Fatal(err)
	}

	tm := parseTimestamp("Mon, 4 Feb 2012 15:44:15 +0000")
	want := &QueueMember{
		CallSID:      "CA386025c9bf5d6052a1d1ea42b4d16662",
		DateEnqueued: tm,
		WaitTime:     30,
		Position:     1,
		URI:          "/2010-04-01/Accounts/AC5ef87.../Queues/QUxxxxxxxxx/Members/CA386025c9bf5d6052a1d1ea42b4d16662.json",
	}

	if !reflect.DeepEqual(queueMember, want) {
		t.Errorf("Queues.Front() returned %+v, want %+v", queueMember, want)
	}

}
