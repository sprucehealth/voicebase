package api

import "time"

const (
	QUEUE_ITEM_STATUS_PENDING                 = "PENDING"
	QUEUE_ITEM_STATUS_COMPLETED               = "TREATED"
	QUEUE_ITEM_STATUS_TRIAGED                 = "TRIAGED"
	QUEUE_ITEM_STATUS_ONGOING                 = "ONGOING"
	QUEUE_ITEM_STATUS_PHOTOS_REJECTED         = "PHOTOS_REJECTED"
	QUEUE_ITEM_STATUS_REFILL_APPROVED         = "APPROVED"
	QUEUE_ITEM_STATUS_REFILL_DENIED           = "DENIED"
	QUEUE_ITEM_STATUS_REPLIED                 = "REPLIED"
	DISPLAY_TYPE_TITLE_SUBTITLE_NONACTIONABLE = "title_subtitle_nonactionable"
	DISPLAY_TYPE_TITLE_SUBTITLE_ACTIONABLE    = "title_subtitle_actionable"
)

type FeedDisplayInterface interface {
	GetTitleAndSubtitle(dataApi DataAPI) (title, subtitle string, err error)
	GetImageUrl() string
	GetActionUrl(dataApi DataAPI) (string, error)
	GetDisplayTypes() []string
	GetTimestamp() *time.Time
}
