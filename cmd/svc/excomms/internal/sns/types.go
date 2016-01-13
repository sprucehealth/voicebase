package sns

import "encoding/json"

type IncomingRawMessageNotification struct {
	// ID represents the id of the raw message in the database
	ID uint64 `json:"id"`
}

func (i *IncomingRawMessageNotification) Marshal() ([]byte, error) {
	return json.Marshal(i)
}
