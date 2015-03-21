package sqs

import (
	"strconv"
	"sync"
	"time"
)

type Mock struct {
	handle   int
	mu       sync.Mutex
	Messages map[string]map[string]string
}

func (s *Mock) newHandle() string {
	s.handle++
	return strconv.Itoa(s.handle)
}

func (s *Mock) SendMessage(queueURL string, delaySeconds int, messageBody string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Messages == nil {
		s.Messages = make(map[string]map[string]string)
	}
	q := s.Messages[queueURL]
	if q == nil {
		q = make(map[string]string)
		s.Messages[queueURL] = q
	}
	q[s.newHandle()] = messageBody
	return nil
}

func (s *Mock) ReceiveMessage(queueURL string, attributes []AttributeName, maxNumberOfMessages, visibilityTimeout, waitTimeSeconds int) ([]*Message, error) {
	s.mu.Lock()

	var msgs []*Message
	if s.Messages != nil && maxNumberOfMessages > 0 {
		if q := s.Messages[queueURL]; q != nil {
			for h, m := range q {
				msgs = append(msgs, &Message{
					MessageID:     h,
					ReceiptHandle: h,
					Body:          m,
				})
				if len(msgs) == maxNumberOfMessages {
					break
				}
			}
		}
	}

	s.mu.Unlock()

	if len(msgs) == 0 && waitTimeSeconds > 0 {
		// Sleep for a bit so we don't create a busy loop. Since this is just for mocking / testing
		// no need to trigger on a new messages.
		time.Sleep(time.Millisecond * 100)
	}
	return msgs, nil
}

func (s *Mock) DeleteMessage(queueURL, receiptHandle string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Messages == nil {
		return nil
	}
	if q := s.Messages[queueURL]; q != nil {
		delete(q, receiptHandle)
	}
	return nil
}

func (s *Mock) GetQueueURL(queueName, queueOwnerAWSAccountID string) (string, error) {
	return queueName, nil
}
