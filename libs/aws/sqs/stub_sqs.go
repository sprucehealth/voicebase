package sqs

import (
	"container/list"
	"strconv"
	"time"
)

type StubSQS struct {
	MsgQueue           map[string]*list.List
	receiptHandleToMsg map[string]map[string]*Message
}

func (s *StubSQS) DeleteMessage(queueUrl, receiptHandle string) error {
	// lookup the message to delete from the right queue
	msgToDelete := s.receiptHandleToMsg[queueUrl][receiptHandle]

	if msgToDelete == nil {
		return nil
	}

	msgQueueForList := s.MsgQueue[queueUrl]

	var next *list.Element
	for e := msgQueueForList.Front(); e != nil; e = next {
		next = e.Next()
		if e.Value.(*Message).MessageId == msgToDelete.MessageId {
			msgQueueForList.Remove(e)
		}
	}
	return nil
}

func (s *StubSQS) GetQueueUrl(queueName, queueOwnerAWSAccountId string) (string, error) {
	return queueName, nil
}

func (s *StubSQS) SendMessage(queueUrl string, delaySeconds int, messageBody string) error {
	if s.MsgQueue == nil {
		s.MsgQueue = make(map[string]*list.List)
	}

	// look up queue, if it does not exist create one
	msgQueueForList := s.MsgQueue[queueUrl]
	if msgQueueForList == nil {
		msgQueueForList = list.New()
		s.MsgQueue[queueUrl] = msgQueueForList
	}

	// create a messsage
	msg := &Message{}
	msg.MessageId = strconv.FormatInt(time.Now().UnixNano(), 10)
	msg.ReceiptHandle = msg.MessageId
	msg.Body = messageBody

	// keep track of receipt handle for msg
	if s.receiptHandleToMsg == nil {
		s.receiptHandleToMsg = make(map[string]map[string]*Message)
	}

	if s.receiptHandleToMsg[queueUrl] == nil {
		s.receiptHandleToMsg[queueUrl] = make(map[string]*Message)
	}
	s.receiptHandleToMsg[queueUrl][msg.ReceiptHandle] = msg

	// push the message to the back of the list
	msgQueueForList.PushBack(msg)

	return nil
}

func (s *StubSQS) ReceiveMessage(queueUrl string, attributes []AttributeName, maxNumberOfMessages, visibilityTimeout, waitTimeSeconds int) ([]*Message, error) {
	// lookup queue
	msgQueueForList := s.MsgQueue[queueUrl]
	if msgQueueForList == nil {
		return nil, nil
	}

	frontItem := msgQueueForList.Front()
	if frontItem == nil {
		return nil, nil
	}

	msg := frontItem.Value.(*Message)
	return []*Message{msg}, nil
}
