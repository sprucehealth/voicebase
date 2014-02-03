package sqs

type AttributeName string

const (
	All                              AttributeName = "All"                              // All values
	SenderId                         AttributeName = "SenderId"                         // the AWS account number (or the IP address, if anonymous access is allowed) of the sender
	SentTimestamp                    AttributeName = "SentTimestamp"                    // the time when the message was sent (epoch time in milliseconds)
	ApproximateReceiveCount          AttributeName = "ApproximateReceiveCount"          // the number of times a message has been received but not deleted
	ApproximateFirstReceiveTimestamp AttributeName = "ApproximateFirstReceiveTimestamp" // the time when the message was first received (epoch time in milliseconds)
)

type listQueuesResponse struct {
	QueueUrls []string `xml:"ListQueuesResult>QueueUrl"`
	RequestId string   `xml:"ResponseMetadata>RequestId"`
}

type getQueueUrlResponse struct {
	Url       string `xml:"GetQueueUrlResult>QueueUrl"`
	RequestId string `xml:"ResponseMetadata>RequestId"`
}

type simpleResponse struct {
	RequestId string `xml:"ResponseMetadata>RequestId"`
}

type Attribute struct {
	Name  AttributeName
	Value string
}

type Message struct {
	MessageId     string
	ReceiptHandle string
	MD5OfBody     string
	Body          string
	Attributes    []Attribute `xml:"Attribute"`
}

type receiveMessageResponse struct {
	Messages  []*Message `xml:"ReceiveMessageResult>Message"`
	RequestId string     `xml:"ResponseMetadata>RequestId"`
}
