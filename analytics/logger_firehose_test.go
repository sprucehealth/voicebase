package analytics

import (
	"errors"
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
		&ServerEvent{Event: "test"},
	})
	test.Equals(t, 1, len(l.batches["serverstream"]))
	test.Equals(t, 0, len(l.batches["clientstream"]))
	test.Equals(t, 0, len(l.batches["defaultstream"]))
	fh.Finish()

	// Second event should flush
	fh.Expect(mock.NewExpectation(fh.PutRecordBatch, &firehose.PutRecordBatchInput{
		DeliveryStreamName: ptr.String("serverstream"),
		Records: []*firehose.Record{
			{Data: []byte(`{"application":"","event":"test","time":"0001-01-01 00:00:00.000"}` + "\n")},
			{Data: []byte(`{"application":"","event":"test2","time":"0001-01-01 00:00:00.000"}` + "\n")},
		},
	}))
	fh.PutRecordBatchOutputs = []*firehose.PutRecordBatchOutput{{}}
	l.writeEvents([]Event{
		&ServerEvent{Event: "test2"},
	})
	test.Equals(t, 0, len(l.batches["serverstream"]))
	fh.Finish()

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
