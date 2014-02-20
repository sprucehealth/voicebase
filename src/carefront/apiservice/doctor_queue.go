package apiservice

import (
	"carefront/api"
	"fmt"
	"net/http"

	"github.com/gorilla/schema"
)

const (
	state_completed = "completed"
	state_pending   = "pending"
)

type DoctorQueueHandler struct {
	DataApi api.DataAPI
}

type DoctorQueueRequestData struct {
	State string `schema:"state"`
}

func (d *DoctorQueueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != HTTP_GET {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData DoctorQueueRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	doctorId, err := d.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor id from account id:"+err.Error())
		return
	}

	var pendingItemsDoctorQueue, completedItemsDoctorQueue []*api.DoctorQueueItem

	if requestData.State == "" || requestData.State == state_pending {
		pendingItemsDoctorQueue, err = d.DataApi.GetPendingItemsInDoctorQueue(doctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor queue for doctor: "+err.Error())
			return
		}
	}

	if requestData.State == "" || requestData.State == state_completed {
		completedItemsDoctorQueue, err = d.DataApi.GetCompletedItemsInDoctorQueue(doctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor queue for doctor: "+err.Error())
			return
		}
	}

	doctorDisplayFeed, err := d.convertDoctorQueueIntoDisplayQueue(pendingItemsDoctorQueue, completedItemsDoctorQueue)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to convert doctor queue into a display feed: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &doctorDisplayFeed)
}

func (d *DoctorQueueHandler) convertDoctorQueueIntoDisplayQueue(pendingItems, completedItems []*api.DoctorQueueItem) (*DisplayFeedTabs, error) {
	var doctorDisplayFeedTabs DisplayFeedTabs

	var pendingOrOngoingDisplayFeed, completedDisplayFeed *DisplayFeed
	doctorDisplayFeedTabs.Tabs = make([]*DisplayFeed, 0)

	if pendingItems != nil {
		pendingOrOngoingDisplayFeed = &DisplayFeed{
			Title: "Pending",
		}
		doctorDisplayFeedTabs.Tabs = append(doctorDisplayFeedTabs.Tabs, pendingOrOngoingDisplayFeed)
	}

	if completedItems != nil {
		completedDisplayFeed = &DisplayFeed{
			Title: "Completed",
		}
		doctorDisplayFeedTabs.Tabs = append(doctorDisplayFeedTabs.Tabs, completedDisplayFeed)
	}

	if len(pendingItems) > 0 {

		// put the first item in the queue into the first section of the display feed
		upcomingVisitSection := &DisplayFeedSection{}
		upcomingVisitSection.Title = "Next Visit"

		pendingItems[0].PositionInQueue = 0
		item, err := converQueueItemToDisplayFeedItem(d.DataApi, pendingItems[0])
		if err != nil {
			return nil, err
		}
		upcomingVisitSection.Items = []*DisplayFeedItem{item}

		nextVisitsSection := &DisplayFeedSection{
			Title: fmt.Sprintf("%d Upcoming Visits", len(pendingItems)-1),
			Items: make([]*DisplayFeedItem, 0),
		}
		for i, doctorQueueItem := range pendingItems[1:] {
			doctorQueueItem.PositionInQueue = i + 1
			item, err = converQueueItemToDisplayFeedItem(d.DataApi, doctorQueueItem)
			if err != nil {
				return nil, err
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
				currentDisplaySection = &DisplayFeedSection{
					Title: day,
					Items: make([]*DisplayFeedItem, 0),
				}
				displaySections = append(displaySections, currentDisplaySection)
				lastSeenDay = day
			}
			displayItem, err := converQueueItemToDisplayFeedItem(d.DataApi, completedItem)
			if err != nil {
				return nil, err
			}
			currentDisplaySection.Items = append(currentDisplaySection.Items, displayItem)
		}
		completedDisplayFeed.Sections = displaySections
	}

	return &doctorDisplayFeedTabs, nil
}
