package doctor_queue

import (
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
)

type DisplayFeedItem struct {
	ID           int64                 `json:"id,string,omitempty"`
	Title        string                `json:"title"`
	Subtitle     string                `json:"subtitle,omitempty"`
	Timestamp    *time.Time            `json:"timestamp,omitempty"`
	ImageURL     *app_url.SpruceAsset  `json:"image_url,omitempty"`
	ActionURL    *app_url.SpruceAction `json:"action_url,omitempty"`
	AuthUrl      *app_url.SpruceAction `json:"auth_url,omitempty"`
	DisplayTypes []string              `json:"display_types,omitempty"`
}

func converQueueItemToDisplayFeedItem(dataAPI api.DataAPI, itemToDisplay api.FeedDisplayInterface) (*DisplayFeedItem, error) {
	title, subtitle, err := itemToDisplay.GetTitleAndSubtitle(dataAPI)
	if err != nil {
		return nil, err
	}

	item := &DisplayFeedItem{
		ID:           itemToDisplay.GetID(),
		Title:        title,
		Subtitle:     subtitle,
		ImageURL:     itemToDisplay.GetImageURL(),
		DisplayTypes: itemToDisplay.GetDisplayTypes(),
		Timestamp:    itemToDisplay.GetTimestamp(),
	}

	item.ActionURL, err = itemToDisplay.ActionURL(dataAPI)
	if err != nil {
		return nil, err
	}

	return item, nil
}
