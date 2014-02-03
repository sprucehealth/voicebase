package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"carefront/libs/aws"
	"carefront/libs/aws/cloudtrail"
	"carefront/libs/aws/s3"
	"carefront/libs/aws/sns"
	"carefront/libs/aws/sqs"
)

var (
	awsAccessKey       = flag.String("aws_access_key", "", "AWS Access Key ID")
	awsSecretKey       = flag.String("aws_secret_key", "", "AWS Secret Key")
	awsRole            = flag.String("aws_role", "", "AWS Role")
	awsRegion          = flag.String("aws_region", "", "AWS Region")
	cloudTrailSQSQueue = flag.String("cloudtrail_sqs_queue", "cloudtrail", "CloudTrail SQS queue name")
)

func startCloudTrailIndexer(es *ElasticSearch) error {
	var auth aws.Auth

	if *awsRole == "" {
		*awsRole = os.Getenv("AWS_ROLE")
	}
	if *awsRole != "" {
		var err error
		auth, err = aws.CredentialsForRole(*awsRole)
		if err != nil {
			return err
		}
	} else {
		keys := aws.Keys{
			AccessKey: *awsAccessKey,
			SecretKey: *awsSecretKey,
		}
		if keys.AccessKey == "" || keys.SecretKey == "" {
			keys = aws.KeysFromEnvironment()
		}
		if keys.AccessKey == "" || keys.SecretKey == "" {
			return errors.New("No AWS credentials or role set")
		}
		auth = keys
	}

	if *awsRegion == "" {
		az, err := aws.GetMetadata(aws.MetadataAvailabilityZone)
		if err != nil {
			return err
		}
		*awsRegion = az[:len(az)-1]
	}

	region, ok := aws.Regions[*awsRegion]
	if !ok {
		return errors.New("Unknown region " + *awsRegion)
	}

	awsCli := &aws.Client{
		Auth: auth,
	}

	s3c := s3.S3{
		Region: region,
		Client: awsCli,
	}

	sq := &sqs.SQS{
		Region: region,
		Client: awsCli,
	}

	queueUrl, err := sq.GetQueueUrl(*cloudTrailSQSQueue, "")
	if err != nil {
		return err
	}

	visibilityTimeout := 120
	waitTimeSeconds := 20
	go func() {
		for {
			msgs, err := sq.ReceiveMessage(queueUrl, nil, 1, visibilityTimeout, waitTimeSeconds)
			if err != nil {
				log.Printf("SQS ReceiveMessage failed: %+v", err)
				time.Sleep(time.Second * 10)
			}
			if len(msgs) == 0 {
				// log.Println("No message received, sleeping")
				time.Sleep(time.Second * 10)
			}
			for _, m := range msgs {
				var note sns.SQSMessage
				if err := json.Unmarshal([]byte(m.Body), &note); err != nil {
					log.Printf("Failed to unmarshal SNS notification from SQS Body: %+v", err)
					time.Sleep(time.Second * 10)
					continue
				}
				var ctNote cloudtrail.SNSNotification
				if err := json.Unmarshal([]byte(note.Message), &ctNote); err != nil {
					log.Printf("Failed to unmarshal CloudTrail notification from SNS message: %+v", err)
					time.Sleep(time.Second * 10)
					continue
				}

				failed := 0
				for _, path := range ctNote.S3ObjectKey {
					rd, err := s3c.GetReader(ctNote.S3Bucket, path)
					if err != nil {
						log.Printf("Failed to fetch log from S3 (%s:%s): %+v", ctNote.S3Bucket, path, err)
						failed++
						continue
					}
					var ct cloudtrail.Log
					dec := json.NewDecoder(rd)
					err = dec.Decode(&ct)
					rd.Close()
					if err != nil {
						log.Printf("Failed to decode CloudTrail json (%s:%s): %+v", ctNote.S3Bucket, path, err)
						failed++
						continue
					}
					for _, rec := range ct.Records {
						idx := fmt.Sprintf("log-%s", rec.EventTime.UTC().Format("2006.01.02"))
						recBytes, err := json.Marshal(rec)
						if err != nil {
							log.Printf("Failed to marshal event: %+v", err)
							failed++
							continue
						}
						recBytes = append(recBytes[:len(recBytes)-1], fmt.Sprintf(`,"@timestamp":"%s","@version":"1","@app":"syslogidx"}`, rec.EventTime.UTC().Format(time.RFC3339))...)
						// log.Printf("%s %s\n", idx, string(recBytes))
						if err := es.IndexJSON(idx, "cloudtrail", recBytes, rec.EventTime); err != nil {
							failed++
							log.Printf("Failed to index event: %+v", err)
							break
						}
					}
					if failed > 0 {
						break
					}
				}
				if failed == 0 {
					if err := sq.DeleteMessage(queueUrl, m.ReceiptHandle); err != nil {
						log.Printf("Failed to delete message: %+v", err)
					}
				}
			}
		}
	}()

	return nil
}
