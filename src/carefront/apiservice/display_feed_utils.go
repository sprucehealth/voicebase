package apiservice

import (
	"carefront/api"
)

type DisplayFeedSection struct {
	Title string             `json:"title"`
	Items []*DisplayFeedItem `json:"items"`
}

type DisplayFeedItem struct {
	Title        string      `json:"title"`
	Subtitle     string      `json:"subtitle"`
	Button       *api.Button `json:"button,omitempty"`
	ImageUrl     string      `json:"image_url"`
	ItemUrl      string      `json:"action_url,omitempty"`
	DisplayTypes []string    `json:"display_types"`
}

type DisplayFeed struct {
	Sections []*DisplayFeedSection `json:"sections,omitempty"`
	Title    string                `json:"title,omitempty"`
}

type DisplayFeedTabs struct {
	Tabs []*DisplayFeed `json:"tabs"`
}

func converQueueItemToDisplayFeedItem(DataApi api.DataAPI, itemToDisplay api.FeedDisplayInterface) (item *DisplayFeedItem, err error) {
	item = &DisplayFeedItem{}
	item.Button = itemToDisplay.GetButton()
	title, subtitle, err := itemToDisplay.GetTitleAndSubtitle(DataApi)
	if err != nil {
		return
	}
	item.Title = title
	item.Subtitle = subtitle
	item.ImageUrl = itemToDisplay.GetImageUrl()
	item.DisplayTypes = itemToDisplay.GetDisplayTypes()
	item.ItemUrl = itemToDisplay.GetActionUrl()
	return
}
