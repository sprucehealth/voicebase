package analytics

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/test"
)

func TestFirehose(t *testing.T) {
	fh := &mock.Firehose{Expector: &mock.Expector{T: t}}
	l, err := NewFirehoseLogger(fh, map[string]string{"client": "clientstream", "server": "serverstream", "": "defaultstream"}, 2, time.Second*5, metrics.NewRegistry())
	test.OK(t, err)

	// First even shouldn't flush as max buffer size is 2
	l.writeEvents([]Event{
		&ServerEvent{Event: "test1"},
	})
	test.Equals(t, 1, len(l.batches["serverstream"]))
	test.Equals(t, 0, len(l.batches["clientstream"]))
	test.Equals(t, 0, len(l.batches["defaultstream"]))
	fh.Finish()

	// Second event should flush
	fh.Expect(mock.NewExpectation(fh.PutRecordBatch, &firehose.PutRecordBatchInput{
		DeliveryStreamName: ptr.String("serverstream"),
		Records: []*firehose.Record{
			{Data: []byte(`{"application":"","event":"test1","time":"0001-01-01 00:00:00.000"}` + "\n")},
			{Data: []byte(`{"application":"","event":"test2","time":"0001-01-01 00:00:00.000"}` + "\n")},
		},
	}))
	fh.PutRecordBatchOutputs = []*firehose.PutRecordBatchOutput{
		{
			RequestResponses: []*firehose.PutRecordBatchResponseEntry{
				{ErrorMessage: nil},
				{ErrorMessage: nil},
			},
		},
	}

	l.writeEvents([]Event{
		&ServerEvent{Event: "test2"},
	})
	test.Equals(t, 0, len(l.batches["serverstream"]))
	fh.Finish()
}

func TestFirehosePutBatchFail(t *testing.T) {
	fh := &mock.Firehose{Expector: &mock.Expector{T: t}}
	l, err := NewFirehoseLogger(fh, map[string]string{"client": "clientstream", "server": "serverstream", "": "defaultstream"}, 2, time.Second*5, metrics.NewRegistry())
	test.OK(t, err)

	// Put failure should maintain batch
	fh.Expect(mock.NewExpectation(fh.PutRecordBatch, &firehose.PutRecordBatchInput{
		DeliveryStreamName: ptr.String("serverstream"),
		Records: []*firehose.Record{
			{Data: []byte(`{"application":"","event":"test3","time":"0001-01-01 00:00:00.000"}` + "\n")},
			{Data: []byte(`{"application":"","event":"test4","time":"0001-01-01 00:00:00.000"}` + "\n")},
		},
	}))
	fh.PutRecordBatchOutputs = []*firehose.PutRecordBatchOutput{nil}
	fh.PutRecordBatchErrs = []error{errors.New("FAIL")}
	l.writeEvents([]Event{
		&ServerEvent{Event: "test3"},
		&ServerEvent{Event: "test4"},
	})
	test.Equals(t, 2, len(l.batches["serverstream"]))
	fh.Finish()
}

func TestFirehosePutItemFail(t *testing.T) {
	fh := &mock.Firehose{Expector: &mock.Expector{T: t}}
	l, err := NewFirehoseLogger(fh, map[string]string{"client": "clientstream", "server": "serverstream", "": "defaultstream"}, 2, time.Second*5, metrics.NewRegistry())
	test.OK(t, err)

	// Partial put failure should maintain items that failed
	fh.Expect(mock.NewExpectation(fh.PutRecordBatch, &firehose.PutRecordBatchInput{
		DeliveryStreamName: ptr.String("serverstream"),
		Records: []*firehose.Record{
			{Data: []byte(`{"application":"","event":"test5","time":"0001-01-01 00:00:00.000"}` + "\n")},
			{Data: []byte(`{"application":"","event":"test6","time":"0001-01-01 00:00:00.000"}` + "\n")},
		},
	}))
	fh.PutRecordBatchOutputs = []*firehose.PutRecordBatchOutput{
		{
			RequestResponses: []*firehose.PutRecordBatchResponseEntry{
				{ErrorMessage: nil},
				{ErrorMessage: ptr.String("Zoinks")},
			},
		},
	}
	l.writeEvents([]Event{
		&ServerEvent{Event: "test5"},
		&ServerEvent{Event: "test6"},
	})
	test.Equals(t, 1, len(l.batches["serverstream"]))
	test.Equals(t, `{"application":"","event":"test6","time":"0001-01-01 00:00:00.000"}`+"\n", l.batches["serverstream"][0].String())
	fh.Finish()
}

func TestFirehoseLargeBatch(t *testing.T) {
	fh := &mock.Firehose{Expector: &mock.Expector{T: t}}
	l, err := NewFirehoseLogger(fh, map[string]string{"client": "clientstream", "server": "serverstream", "": "defaultstream"}, 2, time.Second*5, metrics.NewRegistry())
	test.OK(t, err)

	// Partial put failure should maintain items that failed
	in := &firehose.PutRecordBatchInput{
		DeliveryStreamName: ptr.String("serverstream"),
		Records:            []*firehose.Record{},
	}
	for i := 0; i < maxFirehoseBatchSize; i++ {
		in.Records = append(in.Records, &firehose.Record{
			Data: []byte(fmt.Sprintf(`{"application":"","event":"test%d","time":"0001-01-01 00:00:00.000"}`+"\n", i)),
		})
	}
	fh.Expect(mock.NewExpectation(fh.PutRecordBatch, in))
	out := &firehose.PutRecordBatchOutput{
		RequestResponses: make([]*firehose.PutRecordBatchResponseEntry, maxFirehoseBatchSize),
	}
	for i := 0; i < maxFirehoseBatchSize; i++ {
		out.RequestResponses[i] = &firehose.PutRecordBatchResponseEntry{}
	}
	fh.PutRecordBatchOutputs = []*firehose.PutRecordBatchOutput{out}
	events := make([]Event, maxFirehoseBatchSize+20)
	for i := 0; i < len(events); i++ {
		events[i] = &ServerEvent{Event: fmt.Sprintf("test%d", i)}
	}
	l.writeEvents(events)
	test.Equals(t, 20, len(l.batches["serverstream"]))
	test.Equals(t, fmt.Sprintf(`{"application":"","event":"test%d","time":"0001-01-01 00:00:00.000"}`+"\n", maxFirehoseBatchSize), l.batches["serverstream"][0].String())
	fh.Finish()
}
