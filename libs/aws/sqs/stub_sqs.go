package sqs

import (
	"container/list"
	"strconv"
	"sync"
	"time"
)

type StubSQS struct {
	MsgQueue           map[string]*list.List
	receiptHandleToMsg map[string]map[string]*Message
	mu                 sync.Mutex
}

func (s *StubSQS) DeleteMessage(queueURL, receiptHandle string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// lookup the message to delete from the right queue
	msgToDelete := s.receiptHandleToMsg[queueURL][receiptHandle]

	if msgToDelete == nil {
		return nil
	}

	msgQueueForList := s.MsgQueue[queueURL]

	var next *list.Element
	for e := msgQueueForList.Front(); e != nil; e = next {
		next = e.Next()
		if e.Value.(*Message).MessageID == msgToDelete.MessageID {
			msgQueueForList.Remove(e)
		}
	}
	return nil
}

func (s *StubSQS) GetQueueURL(queueName, queueOwnerAWSAccountId string) (string, error) {
	return queueName, nil
}

func (s *StubSQS) SendMessage(queueURL string, delaySeconds int, messageBody string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.MsgQueue == nil {
		s.MsgQueue = make(map[string]*list.List)
	}

	// look up queue, if it does not exist create one
	msgQueueForList := s.MsgQueue[queueURL]
	if msgQueueForList == nil {
		msgQueueForList = list.New()
		s.MsgQueue[queueURL] = msgQueueForList
	}

	// create a messsage
	msg := &Message{}
	msg.MessageID = strconv.FormatInt(time.Now().UnixNano(), 10)
	msg.ReceiptHandle = msg.MessageID
	msg.Body = messageBody

	// keep track of receipt handle for msg
	if s.receiptHandleToMsg == nil {
		s.receiptHandleToMsg = make(map[string]map[string]*Message)
	}

	if s.receiptHandleToMsg[queueURL] == nil {
		s.receiptHandleToMsg[queueURL] = make(map[string]*Message)
	}
	s.receiptHandleToMsg[queueURL][msg.ReceiptHandle] = msg

	// push the message to the back of the list
	msgQueueForList.PushBack(msg)

	return nil
}

func (s *StubSQS) ReceiveMessage(queueURL string, attributes []AttributeName, maxNumberOfMessages, visibilityTimeout, waitTimeSeconds int) ([]*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// lookup queue
	msgQueueForList := s.MsgQueue[queueURL]
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
