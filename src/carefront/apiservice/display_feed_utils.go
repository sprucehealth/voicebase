package apiservice

import (
	"carefront/api"
	"time"
)

type DisplayFeedSection struct {
	Title string             `json:"title"`
	Items []*DisplayFeedItem `json:"items"`
}

type DisplayFeedItem struct {
	Title        string      `json:"title"`
	Subtitle     string      `json:"subtitle,omitempty"`
	Timestamp    *time.Time  `json:"timestamp,omitempty"`
	Button       *api.Button `json:"button,omitempty"`
	ImageUrl     string      `json:"image_url,omitempty"`
	ItemUrl      string      `json:"action_url,omitempty"`
	DisplayTypes []string    `json:"display_types,omitempty"`
}

type DisplayFeed struct {
	Sections []*DisplayFeedSection `json:"sections,omitempty"`
	Title    string                `json:"title,omitempty"`
}

type DisplayFeedTabs struct {
	Tabs []*DisplayFeed `json:"tabs"`
}

func converQueueItemToDisplayFeedItem(DataApi api.DataAPI, itemToDisplay api.FeedDisplayInterface) (*DisplayFeedItem, error) {
	title, subtitle, err := itemToDisplay.GetTitleAndSubtitle(DataApi)
	if err != nil {
		return nil, err
	}

	item := &DisplayFeedItem{
		Button:       itemToDisplay.GetButton(),
		Title:        title,
		Subtitle:     subtitle,
		ImageUrl:     itemToDisplay.GetImageUrl(),
		DisplayTypes: itemToDisplay.GetDisplayTypes(),
		Timestamp:    itemToDisplay.GetTimestamp(),
	}

	item.ItemUrl, err = itemToDisplay.GetActionUrl(DataApi)
	if err != nil {
		return nil, err
	}

	return item, nil
}
