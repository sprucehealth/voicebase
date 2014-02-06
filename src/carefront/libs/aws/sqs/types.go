package sqs

type AttributeName string

const (
	All                              AttributeName = "All"
	SenderId                         AttributeName = "SenderId"
	SentTimestamp                    AttributeName = "SentTimestamp"
	ApproximateReceiveCount          AttributeName = "ApproximateReceiveCount"
	ApproximateFirstReceiveTimestamp AttributeName = "ApproximateFirstReceiveTimestamp"
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

type sendMessageResponse struct {
	MessageId string `xml:"SendMessageResult>MessageId"`
	MD5OfBody string `xml:"SendMessageResult>MD5OfMessageBody"`
	RequestId string `xml:"ResponseMetadata>RequestId"`
}
