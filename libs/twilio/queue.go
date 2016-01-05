package twilio

type QueueService struct {
	client *Client
}

type QueueIFace interface {
	Get(queueSID string) (*Queue, *Response, error)
	Front(queueSID string) (*QueueMember, *Response, error)
}

type QueueMember struct {
	CallSID      string    `json:"call_sid"`
	DateEnqueued Timestamp `json:"date_enqueued"`
	WaitTime     int       `json:"wait_time"`
	Position     int       `json:"position"`
	URI          string    `json:"uri"`
}

type Queue struct {
	SID             string    `json:"sid"`
	FriendlyName    string    `json:"friendly_name"`
	CurrentSize     uint32    `json:"current_size"`
	AverageWaitTime uint32    `json:"average_wait_time"`
	MaxSize         uint32    `json:"max_size"`
	DateCreated     Timestamp `json:"date_created"`
	DateUpdated     Timestamp `json:"date_updated"`
	URI             string    `json:"uri"`
}

func (q *QueueService) Get(queueSID string) (*Queue, *Response, error) {
	u := q.client.EndPoint("Queues", queueSID)

	req, err := q.client.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	qm := new(Queue)
	resp, err := q.client.Do(req, qm)
	if err != nil {
		return nil, nil, err
	}

	return qm, resp, nil
}

func (q *QueueService) Front(queueSID string) (*QueueMember, *Response, error) {
	u := q.client.EndPoint("Queues", queueSID, "Members", "Front")

	req, err := q.client.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	qm := new(QueueMember)
	resp, err := q.client.Do(req, qm)
	if err != nil {
		return nil, nil, err
	}

	return qm, resp, nil
}
