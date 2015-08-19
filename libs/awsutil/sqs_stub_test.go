package awsutil

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs"
)

func TestSQSStub(t *testing.T) {
	sq := &SQS{}

	queueURL := "qurl"
	msgBody1 := "foo"
	_, err := sq.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    &queueURL,
		MessageBody: &msgBody1,
	})
	if err != nil {
		t.Fatal(err)
	}
	msgBody2 := "var"
	_, err = sq.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    &queueURL,
		MessageBody: &msgBody2,
	})
	if err != nil {
		t.Fatal(err)
	}

	n := int64(3)
	res, err := sq.ReceiveMessage(&sqs.ReceiveMessageInput{QueueUrl: &queueURL, MaxNumberOfMessages: &n})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Messages) != 2 {
		t.Fatalf("Expected 1 message, got %d", len(res.Messages))
	}
	if !((*res.Messages[0].Body == msgBody1 && *res.Messages[1].Body == msgBody2) || (*res.Messages[0].Body == msgBody2 && *res.Messages[1].Body == msgBody1)) {
		t.Fatalf("Unexpected bodies %s, %s", *res.Messages[0].Body, *res.Messages[1].Body)
	}

	handle := "1"
	if _, err := sq.DeleteMessage(&sqs.DeleteMessageInput{QueueUrl: &queueURL, ReceiptHandle: &handle}); err != nil {
		t.Fatal(err)
	}

	res, err = sq.ReceiveMessage(&sqs.ReceiveMessageInput{QueueUrl: &queueURL})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(res.Messages))
	}
	if m := res.Messages[0]; *m.Body != msgBody2 {
		t.Fatalf("Unexpected body %s", *m.Body)
	}
}
