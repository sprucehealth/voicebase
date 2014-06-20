package api

import (
	"carefront/app_url"
	"time"
)

const (
	DISPLAY_TYPE_TITLE_SUBTITLE_NONACTIONABLE = "title_subtitle_nonactionable"
	DISPLAY_TYPE_TITLE_SUBTITLE_ACTIONABLE    = "title_subtitle_actionable"
)

type FeedDisplayInterface interface {
	GetTitleAndSubtitle(dataApi DataAPI) (title, subtitle string, err error)
	GetImageUrl() *app_url.SpruceAsset
	ActionUrl(dataApi DataAPI) (*app_url.SpruceAction, error)
	GetDisplayTypes() []string
	GetTimestamp() *time.Time
}
