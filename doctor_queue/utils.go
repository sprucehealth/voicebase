package doctor_queue

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/api"
)

type doctorQueueItemID struct {
	eventType string
	status    string
	itemID    int64
	doctorID  int64
	queueType api.DoctorQueueType
}

// constructIDFromItem constructs an ID of the form <EventType>:<Status>:<ItemID>:<DoctorID>:<QueueType>
func constructIDFromItem(queueItem *api.DoctorQueueItem) string {
	return fmt.Sprintf("%s:%s:%d:%d:%s", queueItem.EventType, queueItem.Status, queueItem.ItemID, queueItem.DoctorID, queueItem.QueueType.String())
}

// queueItemPartsFromID breaks down the ID into its components expecting the form <EventType>:<Status>:<ItemID>:<DoctorID>
func queueItemPartsFromID(id string) (*doctorQueueItemID, error) {
	parts := strings.Split(id, ":")
	if len(parts) < 4 {
		return nil, fmt.Errorf("doctor_queue: expected at least 4 parts to the id: '%s'", id)
	}

	var err error
	qid := &doctorQueueItemID{
		eventType: parts[0],
		status:    parts[1],
	}
	qid.itemID, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, err
	}
	qid.doctorID, err = strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil, err
	}
	// TODO: making this optional for now to allow cached queues in doctor app to
	// still work as expected. Can remove shortly after pushing to prod.
	if len(parts) > 4 {
		qid.queueType = api.ParseDoctorQueueType(parts[4])
	}

	return qid, nil
}
