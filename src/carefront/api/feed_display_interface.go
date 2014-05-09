package api

import (
	"carefront/app_url"
	"time"
)

const (
	QUEUE_ITEM_STATUS_PENDING                 = "PENDING"
	QUEUE_ITEM_STATUS_COMPLETED               = "TREATED"
	QUEUE_ITEM_STATUS_TRIAGED                 = "TRIAGED"
	QUEUE_ITEM_STATUS_ONGOING                 = "ONGOING"
	QUEUE_ITEM_STATUS_PHOTOS_REJECTED         = "PHOTOS_REJECTED"
	QUEUE_ITEM_STATUS_REFILL_APPROVED         = "APPROVED"
	QUEUE_ITEM_STATUS_REFILL_DENIED           = "DENIED"
	QUEUE_ITEM_STATUS_REPLIED                 = "REPLIED"
	QUEUE_ITEM_STATUS_READ                    = "READ"
	DISPLAY_TYPE_TITLE_SUBTITLE_NONACTIONABLE = "title_subtitle_nonactionable"
	DISPLAY_TYPE_TITLE_SUBTITLE_ACTIONABLE    = "title_subtitle_actionable"
)

type FeedDisplayInterface interface {
	GetTitleAndSubtitle(dataApi DataAPI) (title, subtitle string, err error)
	GetImageUrl() app_url.SpruceUrl
	GetActionUrl(dataApi DataAPI) (app_url.SpruceUrl, error)
	GetDisplayTypes() []string
	GetTimestamp() *time.Time
}
