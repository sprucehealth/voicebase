package sqs

import (
	"os"
	"strings"
	"testing"

	"carefront/libs/aws"
)

func setupTest(t *testing.T) (*SQS, string) {
	keys := aws.KeysFromEnvironment()
	if keys.AccessKey == "" || keys.SecretKey == "" {
		t.Skip("Skipping aws.sqs tests. AWS keys not found in environment.")
	}
	testQueue := os.Getenv("TEST_SQS_QUEUE")
	if testQueue == "" {
		t.Skip("Skipping sqs.TestError. TEST_SQS_QUEUE env variable not set.")
	}

	cli := &aws.Client{
		Auth: keys,
	}
	sqs := &SQS{
		Region: aws.USEast,
		Client: cli,
	}
	if !strings.HasPrefix(testQueue, "http") {
		var err error
		testQueue, err = sqs.GetQueueUrl(testQueue, "")
		if err != nil {
			t.Fatalf("Failed to lookup url for test queue: %+v", err)
		}
	}
	return sqs, testQueue
}

func TestSQS(t *testing.T) {
	sqs, _ := setupTest(t)

	var queueName string
	if queues, err := sqs.ListQueues(""); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("%+v", queues)
		queueName = QueueName(queues[0])
	}
	qUrl, err := sqs.GetQueueUrl(queueName, "")
	if err != nil {
		t.Fatal(err)
	} else {
		t.Logf("Queue %s URL: %s", queueName, qUrl)
	}
	// msgs, err := sqs.ReceiveMessage(qUrl, []AttributeName{All}, 1, 1, 0)
	// if err != nil {
	// 	t.Fatal(err)
	// } else {
	// 	for _, m := range msgs {
	// 		t.Logf("%+v", m)
	// 	}
	// }
}

func TestError(t *testing.T) {
	sqs, testQueue := setupTest(t)

	if !strings.HasPrefix(testQueue, "http") {
		var err error
		testQueue, err = sqs.GetQueueUrl(testQueue, "")
		if err != nil {
			t.Fatalf("Failed to lookup url for test queue: %+v", err)
		}
	}

	if err := sqs.DeleteMessage(testQueue, "XXX"); err == nil {
		t.Fatalf("Expected error from DeleteMessage on invalid handle")
	} else {
		t.Logf("%+v", err)
	}
}
