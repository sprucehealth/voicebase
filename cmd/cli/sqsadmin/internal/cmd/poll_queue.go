package cmd

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"os"

	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/cli/sqsadmin/internal/config"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/ptr"
)

type pollQueueCmd struct {
	cnf *config.Config
	sqs sqsiface.SQSAPI
	kms kmsiface.KMSAPI
}

func NewPollQueueCmd(cnf *config.Config) (Command, error) {
	sqs, err := cnf.SQSClient()
	if err != nil {
		return nil, err
	}
	kms, err := cnf.KMSClient()
	if err != nil {
		return nil, err
	}
	return &pollQueueCmd{
		cnf: cnf,
		sqs: sqs,
		kms: kms,
	}, nil
}

func (c *pollQueueCmd) Run(args []string) error {
	fs := flag.NewFlagSet("list_queues", flag.ExitOnError)
	queueURL := fs.String("queue_url", "", "The queue url to poll")
	kmsKeyARN := fs.String("kms_key_arn", "", "The KMS key to use as the encryption source")
	retry := fs.Bool("retry", true, "If the polling should retry till a message is located")
	base64Encoded := fs.Bool("base64", true, "If the message should be decoded from base64")
	maxMessages := fs.Int64("max_messages", 10, "The maximum number of messages to poll for")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *queueURL == "" {
		*queueURL = prompt(scn, "Queue URL: ")
		if *queueURL == "" {
			return errors.New("Queue URL is required")
		}
	}

	sqsC := c.sqs

	// If a KMS key was provided then assume we need to use it to decrypt
	if *kmsKeyARN != "" {
		var err error
		sqsC, err = awsutil.NewEncryptedSQS(*kmsKeyARN, c.kms, sqsC)
		if err != nil {
			return err
		}
	}

	var maxWaitTime int64 = 20
	pprint("Polling %s - Max Wait Time: %d seconds - retry till message found: %v - content base64 encoded: %v - max messages: %d\n", *queueURL, maxWaitTime, *retry, *base64Encoded, *maxMessages)
	for true {
		resp, err := sqsC.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            queueURL,
			MaxNumberOfMessages: maxMessages,
			VisibilityTimeout:   ptr.Int64(60 * 5),
			WaitTimeSeconds:     ptr.Int64(maxWaitTime),
		})
		if err != nil {
			return err
		}

		pprint("%d messages found\n", len(resp.Messages))
		for _, m := range resp.Messages {
			if *base64Encoded {
				// Hack: Attempt to detect non blob payloads by looking for json encoding
				if *m.Body != "" && (*m.Body)[0] == '{' {
					snsMessage := &awsutil.SNSSQSMessage{}
					if err := json.Unmarshal([]byte(*m.Body), snsMessage); err != nil {
						return err
					}
					msg, err := base64.StdEncoding.DecodeString(snsMessage.Message)
					if err != nil {
						return err
					}
					m.Body = ptr.String(string(msg))
				} else {
					// If it is just a normal sqs message then we can just decode and decrypt
					msg, err := base64.StdEncoding.DecodeString(*m.Body)
					if err != nil {
						return err
					}
					m.Body = ptr.String(string(msg))
				}
			}
			pprint(m.String() + "\n")
		}
		if len(resp.Messages) > 0 || !*retry {
			break
		}
	}
	return nil
}
