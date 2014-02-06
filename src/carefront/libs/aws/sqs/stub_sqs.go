package sqs

import (
	"container/list"
	"fmt"
	"strconv"
	"time"
)

type StubSQS struct {
	msgQueue           map[string]*list.List
	receiptHandleToMsg map[string]map[string]*Message
}

func (s *StubSQS) DeleteMessage(queueUrl, receiptHandle string) error {
	// lookup the message to delete from the right queue
	msgToDelete := s.receiptHandleToMsg[queueUrl][receiptHandle]

	if msgToDelete == nil {
		return nil
	}

	msgQueue := s.msgQueue[queueUrl]

	var next *list.Element
	for e := msgQueue.Front(); e != nil; e = next {
		next = e.Next()
		if e.Value.(*Message).MessageId == msgToDelete.MessageId {
			msgQueue.Remove(e)
		}
	}
	fmt.Printf("Deleted message. List now looks liks %+v", msgQueue)
	return nil
}

func (s *StubSQS) GetQueueUrl(queueName, queueOwnerAWSAccountId string) (string, error) {
	return queueName, nil
}

func (s *StubSQS) SendMessage(queueUrl string, delaySeconds int, messageBody string) error {
	if s.msgQueue == nil {
		s.msgQueue = make(map[string]*list.List)
	}

	// look up queue, if it does not exist create one
	var msgQueue *list.List
	if s.msgQueue[queueUrl] == nil {
		msgQueue = list.New()
		s.msgQueue[queueUrl] = msgQueue
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
	msgQueue.PushBack(msg)

	fmt.Printf("List looks like %+v", msgQueue)

	return nil
}

func (s *StubSQS) ReceiveMessage(queueUrl string, attributes []AttributeName, maxNumberOfMessages, visibilityTimeout, waitTimeSeconds int) ([]*Message, error) {
	// lookup queue
	msgQueue := s.msgQueue[queueUrl]
	if msgQueue == nil {
		return nil, nil
	}

	msg := msgQueue.Front().Value.(*Message)
	msgQueue.Remove(msgQueue.Front())
	return []*Message{msg}, nil
}
