package doctor_queue

import (
	"carefront/api"
	"carefront/app_url"
	"time"
)

type DisplayFeedSection struct {
	Title string             `json:"title"`
	Items []*DisplayFeedItem `json:"items"`
}

type DisplayFeedItem struct {
	Title        string                `json:"title"`
	Subtitle     string                `json:"subtitle,omitempty"`
	Timestamp    *time.Time            `json:"timestamp,omitempty"`
	ImageUrl     *app_url.SpruceAsset  `json:"image_url,omitempty"`
	ActionUrl    *app_url.SpruceAction `json:"action_url,omitempty"`
	DisplayTypes []string              `json:"display_types,omitempty"`
}

type DisplayFeed struct {
	Sections []*DisplayFeedSection `json:"sections,omitempty"`
	Title    string                `json:"title,omitempty"`
}

type DisplayFeedTabs struct {
	Tabs []*DisplayFeed `json:"tabs"`
}

func converQueueItemToDisplayFeedItem(dataApi api.DataAPI, itemToDisplay api.FeedDisplayInterface) (*DisplayFeedItem, error) {
	title, subtitle, err := itemToDisplay.GetTitleAndSubtitle(dataApi)
	if err != nil {
		return nil, err
	}

	item := &DisplayFeedItem{
		Title:        title,
		Subtitle:     subtitle,
		ImageUrl:     itemToDisplay.GetImageUrl(),
		DisplayTypes: itemToDisplay.GetDisplayTypes(),
		Timestamp:    itemToDisplay.GetTimestamp(),
	}

	item.ActionUrl, err = itemToDisplay.ActionUrl(dataApi)
	if err != nil {
		return nil, err
	}

	return item, nil
}
