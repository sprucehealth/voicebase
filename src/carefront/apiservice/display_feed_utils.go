package apiservice

import (
	"carefront/api"
)

type queueSection struct {
	Title string       `json:"title"`
	Items []*queueItem `json:"items"`
}

type queueItem struct {
	Title        string      `json:"title"`
	Subtitle     string      `json:"subtitle"`
	Button       *api.Button `json:"button,omitempty"`
	ImageUrl     string      `json:"image_url"`
	ItemUrl      string      `json:"item_url,omitempty"`
	DisplayTypes []string    `json:"display_types"`
}

type displayFeed struct {
	Sections []*queueSection `json:"sections,omitempty"`
	Title    string          `json:"title,omitempty"`
}

type displayFeedTabs struct {
	Tabs []*displayFeed `json:"tabs"`
}

func converQueueItemToDisplayFeedItem(DataApi api.DataAPI, itemToDisplay api.FeedDisplayInterface) (item *queueItem, err error) {
	item = &queueItem{}
	item.Button = itemToDisplay.GetButton()
	title, subtitle, err := itemToDisplay.GetTitleAndSubtitle(DataApi)
	if err != nil {
		return
	}
	item.Title = title
	item.Subtitle = subtitle
	item.ImageUrl = itemToDisplay.GetImageUrl()
	item.DisplayTypes = itemToDisplay.GetDisplayTypes()
	return
}
