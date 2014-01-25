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
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor id from account id ")
		return
	}

	pendingItemsDoctorQueue, err := d.DataApi.GetPendingItemsInDoctorQueue(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor queue for doctor ")
		return
	}

	completedItemsDoctorQueue, err := d.DataApi.GetCompletedItemsInDoctorQueue(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor queue for doctor")
		return
	}

	doctorDisplayFeed, err := d.convertDoctorQueueIntoDisplayQueue(pendingItemsDoctorQueue, completedItemsDoctorQueue)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to convert doctor queue into a display feed: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &doctorDisplayFeed)
}

func (d *DoctorQueueHandler) convertDoctorQueueIntoDisplayQueue(pendingItems, completedItems []*api.DoctorQueueItem) (doctorDisplayFeedTabs *DisplayFeedTabs, err error) {
	doctorDisplayFeedTabs = &DisplayFeedTabs{}
	pendingOrOngoingDisplayFeed := &DisplayFeed{}
	pendingOrOngoingDisplayFeed.Title = "Pending"
	completedDisplayFeed := &DisplayFeed{}
	completedDisplayFeed.Title = "Completed"
	doctorDisplayFeedTabs.Tabs = []*DisplayFeed{pendingOrOngoingDisplayFeed, completedDisplayFeed}

	if len(pendingItems) > 0 {
		// put the first item in the queue into the first section of the display feed
		upcomingVisitSection := &DisplayFeedSection{}
		upcomingVisitSection.Title = "Next Visit"

		pendingItems[0].PositionInQueue = 0
		item, shadowedErr := converQueueItemToDisplayFeedItem(d.DataApi, pendingItems[0])
		if shadowedErr != nil {
			err = shadowedErr
			return
		}
		upcomingVisitSection.Items = []*DisplayFeedItem{item}

		nextVisitsSection := &DisplayFeedSection{}
		nextVisitsSection.Title = fmt.Sprintf("%d Upcoming Visits", len(pendingItems)-1)
		nextVisitsSection.Items = make([]*DisplayFeedItem, 0)
		for i, doctorQueueItem := range pendingItems[1:] {
			doctorQueueItem.PositionInQueue = i + 1
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
		displaySections := make([]*DisplayFeedSection, 0)
		currentDisplaySection := &DisplayFeedSection{}
		lastSeenDay := ""
		for i, completedItem := range completedItems {
			completedItem.PositionInQueue = i
			day := fmt.Sprintf("%s %d %d", completedItem.EnqueueDate.Month().String(), completedItem.EnqueueDate.Day(), completedItem.EnqueueDate.Year())
			if lastSeenDay != day {
				currentDisplaySection = &DisplayFeedSection{}
				currentDisplaySection.Title = day
				currentDisplaySection.Items = make([]*DisplayFeedItem, 0)
				displaySections = append(displaySections, currentDisplaySection)
				lastSeenDay = day
			}
			displayItem, shadowedErr := converQueueItemToDisplayFeedItem(d.DataApi, completedItem)
			if shadowedErr != nil {
				err = shadowedErr
				return
			}
			currentDisplaySection.Items = append(currentDisplaySection.Items, displayItem)
		}
		completedDisplayFeed.Sections = displaySections
	}

	return
}
