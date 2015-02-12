package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/libs/aws/cloudtrail"
	"github.com/sprucehealth/backend/libs/aws/sns"
	"github.com/sprucehealth/backend/libs/aws/sqs"
	"github.com/sprucehealth/backend/libs/golog"
)

var (
	cloudTrailSQSQueue = flag.String("cloudtrail_sqs_queue", "cloudtrail", "CloudTrail SQS queue name")
)

/*
CloudTrail is configured to write logs to an S3 bucket and post a notification
to an SNS topic when a new log is written. The SNS topic is setup to enqueue a
message in SQS for each notification.

To index the log we receive message from the SQS queue, pull down the log from
S3, and then parse and index the events in ElasticSearch. Only after the events
have successfully been indexed do we delete the message from the SQS queue. It's
possible some events may get indexed multiple times (due to partial success),
but this is more desirable than missing out on events. It may be possible to
generate a unique ID for each event to avoid this (e.g. hash of log file
key + record index).
*/

func startCloudTrailIndexer(es *ElasticSearch) error {
	sq := &sqs.SQS{
		Region: region,
		Client: awsClient,
	}

	queueURL, err := sq.GetQueueURL(*cloudTrailSQSQueue, "")
	if err != nil {
		return err
	}

	visibilityTimeout := 120
	waitTimeSeconds := 20
	go func() {
		for {
			msgs, err := sq.ReceiveMessage(queueURL, nil, 1, visibilityTimeout, waitTimeSeconds)
			if err != nil {
				golog.Errorf("SQS ReceiveMessage failed: %+v", err)
				time.Sleep(time.Minute)
				continue
			}
			if len(msgs) == 0 {
				// log.Println("No message received, sleeping")
				time.Sleep(time.Minute)
				continue
			}
			for _, m := range msgs {
				var note sns.SQSMessage
				if err := json.Unmarshal([]byte(m.Body), &note); err != nil {
					golog.Errorf("Failed to unmarshal SNS notification from SQS Body: %+v", err)
					continue
				}
				var ctNote cloudtrail.SNSNotification
				if err := json.Unmarshal([]byte(note.Message), &ctNote); err != nil {
					golog.Errorf("Failed to unmarshal CloudTrail notification from SNS message: %+v", err)
					continue
				}

				failed := 0
				for _, path := range ctNote.S3ObjectKey {
					rd, _, err := s3Client.GetReader(ctNote.S3Bucket, path)
					if err != nil {
						golog.Errorf("Failed to fetch log from S3 (%s:%s): %+v", ctNote.S3Bucket, path, err)
						failed++
						continue
					}
					var ct cloudtrail.Log
					err = json.NewDecoder(rd).Decode(&ct)
					rd.Close()
					if err != nil {
						golog.Errorf("Failed to decode CloudTrail json (%s:%s): %+v", ctNote.S3Bucket, path, err)
						failed++
						continue
					}
					for _, rec := range ct.Records {
						ts := rec.EventTime.UTC()
						idx := fmt.Sprintf("log-%s", ts.Format("2006.01.02"))

						doc := map[string]interface{}{
							"@timestamp":        ts.Format(time.RFC3339),
							"@version":          "1",
							"awsRegion":         rec.AWSRegion,
							"errorCode":         rec.ErrorCode,
							"errorMessage":      rec.ErrorMessage,
							"eventName":         rec.EventName,
							"eventSource":       rec.EventSource,
							"eventTime":         rec.EventTime,
							"eventVersion":      rec.EventVersion,
							"requestParameters": rec.RequestParameters,
							"responseElements":  rec.ResponseElements,
							"sourceIPAddress":   rec.SourceIPAddress,
							"userAgent":         rec.UserAgent,
							"userIdentity":      rec.UserIdentity,
						}

						docBytes, err := json.Marshal(doc)
						if err != nil {
							golog.Errorf("Failed to marshal event: %+v", err)
							failed++
							continue
						}

						if err := es.IndexJSON(idx, "cloudtrail", docBytes, ts); err != nil {
							failed++
							golog.Errorf("Failed to index event: %+v", err)
							break
						}
					}
					if failed > 0 {
						break
					}
				}
				if failed == 0 {
					if err := sq.DeleteMessage(queueURL, m.ReceiptHandle); err != nil {
						golog.Errorf("Failed to delete message: %+v", err)
					}
				}
			}
		}
	}()

	return nil
}
