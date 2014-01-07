package apiservice

import (
	"carefront/api"
	"fmt"
	"net/http"
)

type DoctorQueueHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

func (d *DoctorQueueHandler) AccountIdFromAuthToken(accountId int64) {
	d.accountId = accountId
}

func (d *DoctorQueueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorId, err := d.DataApi.GetDoctorIdFromAccountId(d.accountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor id from account id "+err.Error())
		return
	}

	doctorQueue, err := d.DataApi.GetDoctorQueue(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor queue for doctor : "+err.Error())
		return
	}

	doctorDisplayFeed, err := d.convertDoctorQueueIntoDisplayQueue(doctorQueue)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to convert doctor queue into a display feed: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &doctorDisplayFeed)
}

func (d *DoctorQueueHandler) convertDoctorQueueIntoDisplayQueue(doctorQueue []*api.DoctorQueueItem) (doctorDisplayFeedTabs *displayFeedTabs, err error) {
	doctorDisplayFeedTabs = &displayFeedTabs{}
	pendingOrOngoingDisplayFeed := &displayFeed{}
	pendingOrOngoingDisplayFeed.Title = "Pending"
	completedDisplayFeed := &displayFeed{}
	completedDisplayFeed.Title = "Completed"
	doctorDisplayFeedTabs.Tabs = []*displayFeed{pendingOrOngoingDisplayFeed, completedDisplayFeed}

	pendingOrOngoingItems := make([]*api.DoctorQueueItem, 0)
	completedItems := make([]*api.DoctorQueueItem, 0)
	for _, queueItem := range doctorQueue {
		switch queueItem.Status {
		case api.QUEUE_ITEM_STATUS_PENDING, api.QUEUE_ITEM_STATUS_ONGOING:
			pendingOrOngoingItems = append(pendingOrOngoingItems, queueItem)
		case api.QUEUE_ITEM_STATUS_COMPLETED:
			completedItems = append(completedItems, queueItem)
		}
	}

	if len(pendingOrOngoingItems) > 0 {
		// put the first item in the queue into the first section of the display feed
		upcomingVisitSection := &queueSection{}
		upcomingVisitSection.Title = "Next Visit"
		item, shadowedErr := converQueueItemToDisplayFeedItem(d.DataApi, pendingOrOngoingItems[0])
		if shadowedErr != nil {
			err = shadowedErr
			return
		}
		upcomingVisitSection.Items = []*queueItem{item}

		nextVisitsSection := &queueSection{}
		nextVisitsSection.Title = fmt.Sprintf("%d Upcoming Visits", len(pendingOrOngoingItems)-1)
		nextVisitsSection.Items = make([]*queueItem, 0)
		for _, doctorQueueItem := range pendingOrOngoingItems[1:] {
			item, err = converQueueItemToDisplayFeedItem(d.DataApi, doctorQueueItem)
			if err != nil {
				return
			}
			nextVisitsSection.Items = append(nextVisitsSection.Items, item)
		}

		pendingOrOngoingDisplayFeed.Sections = []*queueSection{upcomingVisitSection, nextVisitsSection}
	}

	if len(completedItems) > 0 {
		// cluster feed items based on day
		itemsByDay := make(map[string][]*queueItem)
		for _, completedItem := range completedItems {
			day := fmt.Sprintf("%s %d %d", completedItem.EnqueueDate.Month().String(), completedItem.EnqueueDate.Day(), completedItem.EnqueueDate.Year())
			itemsList := itemsByDay[day]
			if itemsList == nil {
				itemsList = make([]*queueItem, 0)
			}
			displayItem, shadowedErr := converQueueItemToDisplayFeedItem(d.DataApi, completedItem)
			if shadowedErr != nil {
				err = shadowedErr
				return
			}
			itemsList = append(itemsList, displayItem)
			itemsByDay[day] = itemsList
		}

		displaySections := make([]*queueSection, 0)
		for sectionTitle, itemsList := range itemsByDay {
			displaySection := &queueSection{}
			displaySection.Title = sectionTitle
			displaySection.Items = itemsList
			displaySections = append(displaySections, displaySection)
		}

		completedDisplayFeed.Sections = displaySections
	}

	return
}
