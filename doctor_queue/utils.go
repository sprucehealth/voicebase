package doctor_queue

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/api"
)

// constructIDFromItem constructs an ID of the form <EventType>:<Status>:<ItemID>:<DoctorID>
func constructIDFromItem(queueItem *api.DoctorQueueItem) string {
	return fmt.Sprintf("%s:%s:%d:%d", queueItem.EventType, queueItem.Status, queueItem.ItemID, queueItem.DoctorID)
}

// queueItemPartsFromID breaks down the ID into its components expecting the form <EventType>:<Status>:<ItemID>:<DoctorID>
func queueItemPartsFromID(id string) (eventType, status string, itemID, doctorID int64, err error) {
	parts := strings.Split(id, ":")
	if len(parts) != 4 {
		err = fmt.Errorf("Expected 4 parts to the id, got %d", len(parts))
		return
	}

	eventType = parts[0]
	status = parts[1]
	itemID, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}
	doctorID, err = strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return
	}

	return
}
