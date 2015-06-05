package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/golog"
)

var (
	cloudTrailSQSQueue = flag.String("cloudtrail_sqs_queue", "cloudtrail", "CloudTrail SQS queue name")
)

func startCloudTrailIndexer(es *ElasticSearch) error {
	sq := sqs.New(awsConfig)

	res, err := sq.GetQueueURL(&sqs.GetQueueURLInput{QueueName: cloudTrailSQSQueue})
	if err != nil {
		return err
	}
	queueURL := *res.QueueURL

	visibilityTimeout := int64(120)
	waitTimeSeconds := int64(20)
	go func() {
		for {
			res, err := sq.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueURL:          &queueURL,
				VisibilityTimeout: &visibilityTimeout,
				WaitTimeSeconds:   &waitTimeSeconds,
			})
			if err != nil {
				golog.Errorf("SQS ReceiveMessage failed: %+v", err)
				time.Sleep(time.Minute)
				continue
			}
			if len(res.Messages) == 0 {
				// log.Println("No message received, sleeping")
				time.Sleep(time.Minute)
				continue
			}
			for _, m := range res.Messages {
				var note awsutil.SNSSQSMessage
				if err := json.Unmarshal([]byte(*m.Body), &note); err != nil {
					golog.Errorf("Failed to unmarshal SNS notification from SQS Body: %+v", err)
					continue
				}
				var ctNote awsutil.CloudTrailSNSNotification
				if err := json.Unmarshal([]byte(note.Message), &ctNote); err != nil {
					golog.Errorf("Failed to unmarshal CloudTrail notification from SNS message: %+v", err)
					continue
				}

				failed := 0
				for _, path := range ctNote.S3ObjectKey {
					res, err := s3Client.GetObject(&s3.GetObjectInput{
						Bucket: &ctNote.S3Bucket,
						Key:    &path,
					})
					if err != nil {
						golog.Errorf("Failed to fetch log from S3 (%s:%s): %+v", ctNote.S3Bucket, path, err)
						failed++
						continue
					}
					rd := res.Body
					var ct awsutil.CloudTrailLog
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
					_, err := sq.DeleteMessage(&sqs.DeleteMessageInput{
						QueueURL:      &queueURL,
						ReceiptHandle: m.ReceiptHandle,
					})
					if err != nil {
						golog.Errorf("Failed to delete message: %+v", err)
					}
				}
			}
		}
	}()

	return nil
}
