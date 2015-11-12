package analytics

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	defaultFirehoseMaxBatchSize     = 128
	defaultfirehoseMaxBatchDuration = time.Second * 5
)

var (
	bufferPool = sync.Pool{}
	recordPool = sync.Pool{}
)

// FirehoseLogger is an analytics logger that sends events to
// AWS Kinesis Firehose as JSON encoded strings.
type FirehoseLogger struct {
	fh               firehoseiface.FirehoseAPI
	categoryToStream map[string]string
	maxBatchSize     int
	maxBatchDuration time.Duration
	backoff          time.Duration              // duration to wait before retry on failure
	batches          map[string][]*bytes.Buffer // stream -> serialized event buffers
	lastPut          map[string]time.Time       // stream -> last put time
	eventCh          chan []Event
	statEvents       *metrics.Counter
	statPutBatchSize metrics.Histogram
	statPutSuccess   *metrics.Counter
	statPutFailure   *metrics.Counter
	statBatchSize    *metrics.IntegerGauge
}

// NewFirehoseLogger returns a new instance of FirehoseLogger. maxBatchSize sets
// the maximum number of events to batch before transmitting. maxBatchDuration
// sets the maximum time to batch events. categoryToStream is a mapping of event
// category to firehose stream name. If it includes an empty string key then
// the matching stream will be used as the default for all events with a
// category without an explicit stream.
func NewFirehoseLogger(fh firehoseiface.FirehoseAPI, categoryToStream map[string]string, maxBatchSize int, maxBatchDuration time.Duration, metricsRegistry metrics.Registry) (*FirehoseLogger, error) {
	if fh == nil {
		return nil, errors.Trace(errors.New("analytics.firehose: firehose API is required"))
	}
	if len(categoryToStream) == 0 {
		return nil, errors.Trace(errors.New("analytics.firehose: at least one category -> stream mapping is required"))
	}
	if maxBatchSize <= 0 {
		maxBatchSize = defaultFirehoseMaxBatchSize
	}
	if maxBatchDuration <= 0 {
		maxBatchDuration = defaultfirehoseMaxBatchDuration
	}
	l := &FirehoseLogger{
		fh:               fh,
		categoryToStream: categoryToStream,
		maxBatchSize:     maxBatchSize,
		maxBatchDuration: maxBatchDuration,
		batches:          make(map[string][]*bytes.Buffer),
		lastPut:          make(map[string]time.Time),
		statEvents:       metrics.NewCounter(),
		statPutBatchSize: metrics.NewUnbiasedHistogram(),
		statPutSuccess:   metrics.NewCounter(),
		statPutFailure:   metrics.NewCounter(),
		statBatchSize:    metrics.NewIntegerGauge(),
	}
	metricsRegistry.Add("batchsize", l.statBatchSize)
	metricsRegistry.Add("events", l.statEvents)
	metricsRegistry.Add("put/batchsize", l.statPutBatchSize)
	metricsRegistry.Add("put/success", l.statPutSuccess)
	metricsRegistry.Add("put/failure", l.statPutFailure)
	for _, stream := range categoryToStream {
		l.lastPut[stream] = time.Now()
	}
	return l, nil
}

// Start opens the firehose
func (l *FirehoseLogger) Start() error {
	return l.startWithBuffer(eventBufferSize)
}

// Stop flushes any buffered events and closes the firehose
func (l *FirehoseLogger) Stop() error {
	close(l.eventCh)
	return nil
}

// WriteEvents buffers events to be written to the firehose
func (l *FirehoseLogger) WriteEvents(events []Event) {
	l.eventCh <- events
}

func (l *FirehoseLogger) startWithBuffer(n int) error {
	l.eventCh = make(chan []Event, n)
	go l.loop()
	return nil
}

func (l *FirehoseLogger) loop() {
	// Periodically submit an emtpy events slice to force a check for need to flush
	tc := time.NewTicker(time.Second)
	defer tc.Stop()
	stopCh := make(chan struct{})
	defer close(stopCh)
	go func() {
		for {
			select {
			case <-tc.C:
				select {
				case l.eventCh <- nil:
				default:
				}
			case <-stopCh:
				return
			}
		}
	}()
	for ev := range l.eventCh {
		l.writeEvents(ev)
	}
	for stream, batch := range l.batches {
		l.batches[stream] = l.flush(stream, batch)
	}
}

func (l *FirehoseLogger) writeEvents(events []Event) {
	l.statEvents.Inc(uint64(len(events)))
	for _, e := range events {
		cat := e.Category()
		stream := l.categoryToStream[cat]
		if stream == "" {
			stream = l.categoryToStream[""]
			if stream == "" {
				continue
			}
		}

		var buf *bytes.Buffer
		if b := bufferPool.Get(); b != nil {
			buf = b.(*bytes.Buffer)
			buf.Reset()
		} else {
			buf = &bytes.Buffer{}
		}
		if err := json.NewEncoder(buf).Encode(e); err != nil {
			golog.Errorf("analytics.firehose: failed to encode event: %s", err)
		} else {
			l.batches[stream] = append(l.batches[stream], buf)
		}
	}
	now := time.Now()
	total := 0
	for stream, batch := range l.batches {
		if len(batch) > 0 && (len(batch) >= l.maxBatchSize || now.Sub(l.lastPut[stream]) > l.maxBatchDuration) && (now.Sub(l.lastPut[stream]) > l.backoff) {
			l.lastPut[stream] = now
			l.batches[stream] = l.flush(stream, batch)
		}
		total += len(l.batches[stream])
	}
	l.statBatchSize.Set(int64(total))
}

func (l *FirehoseLogger) flush(stream string, batch []*bytes.Buffer) []*bytes.Buffer {
	if len(batch) == 0 {
		return batch
	}
	inp := &firehose.PutRecordBatchInput{
		DeliveryStreamName: &stream,
		Records:            make([]*firehose.Record, len(batch)),
	}
	for i, b := range batch {
		var rec *firehose.Record
		if r := recordPool.Get(); r != nil {
			rec = r.(*firehose.Record)
			rec.Data = b.Bytes()
		} else {
			rec = &firehose.Record{
				Data: b.Bytes(),
			}
		}
		inp.Records[i] = rec
	}
	l.statPutBatchSize.Update(int64(len(inp.Records)))
	outp, err := l.fh.PutRecordBatch(inp)
	if err != nil {
		l.statPutFailure.Inc(1)
		golog.Errorf("analytics.firehose: PutRecordBatch failed: %s", err)
		l.backoff = time.Second * 10
	} else {
		l.statPutSuccess.Inc(1)
		l.backoff = 0
		for i, r := range outp.RequestResponses {
			if r.ErrorMessage != nil && *r.ErrorMessage != "" {
				golog.Errorf("analytics.firehose: failed to put event '%s': %s", string(inp.Records[i].Data), *r.ErrorMessage)
			}
		}
		// Recycle buffers
		for i, b := range batch {
			bufferPool.Put(b)
			batch[i] = nil // Clear the pointer to allow the buffer to be GCd if the pool is cleared
		}
		batch = batch[:0]
	}
	// Recycle records
	for _, r := range inp.Records {
		r.Data = nil
		recordPool.Put(r)
	}
	return batch
}
