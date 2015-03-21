package sqs

type AttributeName string

const (
	All                              AttributeName = "All"                              // All values
	SenderID                         AttributeName = "SenderId"                         // the AWS account number (or the IP address, if anonymous access is allowed) of the sender
	SentTimestamp                    AttributeName = "SentTimestamp"                    // the time when the message was sent (epoch time in milliseconds)
	ApproximateReceiveCount          AttributeName = "ApproximateReceiveCount"          // the number of times a message has been received but not deleted
	ApproximateFirstReceiveTimestamp AttributeName = "ApproximateFirstReceiveTimestamp" // the time when the message was first received (epoch time in milliseconds)
)

type listQueuesResponse struct {
	QueueUrls []string `xml:"ListQueuesResult>QueueUrl"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type getQueueUrlResponse struct {
	Url       string `xml:"GetQueueUrlResult>QueueUrl"`
	RequestID string `xml:"ResponseMetadata>RequestId"`
}

type simpleResponse struct {
	RequestID string `xml:"ResponseMetadata>RequestId"`
}

type Attribute struct {
	Name  AttributeName
	Value string
}

type Message struct {
	MessageID     string `xml:"MessageId"`
	ReceiptHandle string
	MD5OfBody     string
	Body          string
	Attributes    []Attribute `xml:"Attribute"`
}

type receiveMessageResponse struct {
	Messages  []*Message `xml:"ReceiveMessageResult>Message"`
	RequestID string     `xml:"ResponseMetadata>RequestId"`
}

type sendMessageResponse struct {
	MessageID string `xml:"SendMessageResult>MessageId"`
	MD5OfBody string `xml:"SendMessageResult>MD5OfMessageBody"`
	RequestID string `xml:"ResponseMetadata>RequestId"`
}

type SQSService interface {
	DeleteMessage(queueURL, receiptHandle string) error
	GetQueueURL(queueName, queueOwnerAWSAccountID string) (string, error)
	SendMessage(queueURL string, delaySeconds int, messageBody string) error
	ReceiveMessage(queueURL string, attributes []AttributeName, maxNumberOfMessages, visibilityTimeout, waitTimeSeconds int) ([]*Message, error)
}
