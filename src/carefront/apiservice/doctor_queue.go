package apiservice

import (
	"carefront/api"
	"fmt"
	"net/http"
)

type DoctorQueueHandler struct {
	DataApi api.DataAPI
}

func (d *DoctorQueueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorId, err := d.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
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

func (d *DoctorQueueHandler) convertDoctorQueueIntoDisplayQueue(doctorQueue []*api.DoctorQueueItem) (doctorDisplayFeedTabs *DisplayFeedTabs, err error) {
	doctorDisplayFeedTabs = &DisplayFeedTabs{}
	pendingOrOngoingDisplayFeed := &DisplayFeed{}
	pendingOrOngoingDisplayFeed.Title = "Pending"
	completedDisplayFeed := &DisplayFeed{}
	completedDisplayFeed.Title = "Completed"
	doctorDisplayFeedTabs.Tabs = []*DisplayFeed{pendingOrOngoingDisplayFeed, completedDisplayFeed}

	pendingOrOngoingItems := make([]*api.DoctorQueueItem, 0)
	completedItems := make([]*api.DoctorQueueItem, 0)

	// first go through and populate all the ongoing items to give them priority
	for _, queueItem := range doctorQueue {
		switch queueItem.Status {
		case api.QUEUE_ITEM_STATUS_ONGOING:
			pendingOrOngoingItems = append(pendingOrOngoingItems, queueItem)
		}
	}

	// then go through and populate all the pending or completed items
	for _, queueItem := range doctorQueue {
		switch queueItem.Status {
		case api.QUEUE_ITEM_STATUS_PENDING:
			pendingOrOngoingItems = append(pendingOrOngoingItems, queueItem)
		case api.QUEUE_ITEM_STATUS_COMPLETED, api.QUEUE_ITEM_STATUS_PHOTOS_REJECTED:
			completedItems = append(completedItems, queueItem)
		}
	}

	if len(pendingOrOngoingItems) > 0 {
		// put the first item in the queue into the first section of the display feed
		upcomingVisitSection := &DisplayFeedSection{}
		upcomingVisitSection.Title = "Next Visit"
		item, shadowedErr := converQueueItemToDisplayFeedItem(d.DataApi, pendingOrOngoingItems[0])
		if shadowedErr != nil {
			err = shadowedErr
			return
		}
		upcomingVisitSection.Items = []*DisplayFeedItem{item}

		nextVisitsSection := &DisplayFeedSection{}
		nextVisitsSection.Title = fmt.Sprintf("%d Upcoming Visits", len(pendingOrOngoingItems)-1)
		nextVisitsSection.Items = make([]*DisplayFeedItem, 0)
		for _, doctorQueueItem := range pendingOrOngoingItems[1:] {
			item, err = converQueueItemToDisplayFeedItem(d.DataApi, doctorQueueItem)
			if err != nil {
				return
			}
			nextVisitsSection.Items = append(nextVisitsSection.Items, item)
		}

		pendingOrOngoingDisplayFeed.Sections = []*DisplayFeedSection{upcomingVisitSection, nextVisitsSection}
	}

	if len(completedItems) > 0 {
		// cluster feed items based on day
		itemsByDay := make(map[string][]*DisplayFeedItem)
		for _, completedItem := range completedItems {
			day := fmt.Sprintf("%s %d %d", completedItem.EnqueueDate.Month().String(), completedItem.EnqueueDate.Day(), completedItem.EnqueueDate.Year())
			itemsList := itemsByDay[day]
			if itemsList == nil {
				itemsList = make([]*DisplayFeedItem, 0)
			}
			displayItem, shadowedErr := converQueueItemToDisplayFeedItem(d.DataApi, completedItem)
			if shadowedErr != nil {
				err = shadowedErr
				return
			}
			itemsList = append(itemsList, displayItem)
			itemsByDay[day] = itemsList
		}

		displaySections := make([]*DisplayFeedSection, 0)
		for sectionTitle, itemsList := range itemsByDay {
			displaySection := &DisplayFeedSection{}
			displaySection.Title = sectionTitle
			displaySection.Items = itemsList
			displaySections = append(displaySections, displaySection)
		}

		completedDisplayFeed.Sections = displaySections
	}

	return
}
