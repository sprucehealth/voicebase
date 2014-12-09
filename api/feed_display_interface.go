package api

import (
	"time"

	"github.com/sprucehealth/backend/app_url"
)

const (
	DisplayTypeTitleSubtitleActionable = "title_subtitle_actionable"
)

type FeedDisplayInterface interface {
	GetID() int64
	GetTitleAndSubtitle(dataAPI DataAPI) (title, subtitle string, err error)
	GetImageURL() *app_url.SpruceAsset
	ActionURL(dataAPI DataAPI) (*app_url.SpruceAction, error)
	GetDisplayTypes() []string
	GetTimestamp() *time.Time
}
