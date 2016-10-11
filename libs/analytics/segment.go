package analytics

import (
	analytics "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
)

var segmentClient *analytics.Client

func InitSegment(key string) {

	if segmentClient != nil {
		panic("segment client already initialized")
	}

	segmentClient = analytics.New(key)
}

// AnalyticsOption allows you to specify options for analytic events that are being posted.
type AnalyticsOption int

const (
	// Synchronous indicates that the analytics event should be posted synchronously.
	Synchronous AnalyticsOption = iota + 1
)

type analyticsOptions []AnalyticsOption

func (qos analyticsOptions) Has(opt AnalyticsOption) bool {
	for _, o := range qos {
		if o == opt {
			return true
		}
	}
	return false
}

// SegmentAlias calls the Alias API using a segment client to
// merge two user identities.
// https://segment.com/docs/spec/alias/
func SegmentAlias(msg *analytics.Alias, opts ...AnalyticsOption) {
	if segmentClient == nil {
		golog.Infof("SegmentIO Alias(%+v)", msg)
		return
	}

	f := segmentClient.Alias

	if analyticsOptions(opts).Has(Synchronous) {
		if err := f(msg); err != nil {
			golog.Errorf("SegmentIO Alias(%+v) failed: %s", msg, err)
		}
	} else {
		conc.Go(func() {
			if err := f(msg); err != nil {
				golog.Errorf("SegmentIO Alias(%+v) failed: %s", msg, err)
			}
		})
	}
}

// SegmentGroup calls the Group API using a segment client
// to associate an individual user with a group.
// https://segment.com/docs/spec/group/
func SegmentGroup(msg *analytics.Group, opts ...AnalyticsOption) {
	if segmentClient == nil {
		golog.Infof("SegmentIO Group(%+v)", msg)
		return
	}

	f := segmentClient.Group
	if analyticsOptions(opts).Has(Synchronous) {
		if err := f(msg); err != nil {
			golog.Errorf("SegmentIO Group(%+v) failed: %s", msg, err)
		}
	} else {
		conc.Go(func() {
			if err := f(msg); err != nil {
				golog.Errorf("SegmentIO Group(%+v) failed: %s", msg, err)
			}
		})
	}
}

// SegmentIdentify calls the Identify API using a segment client
// that ties a customer and their actions to a recognizable ID
// https://segment.com/docs/spec/identify/
func SegmentIdentify(msg *analytics.Identify, opts ...AnalyticsOption) {
	if segmentClient == nil {
		golog.Infof("SegmentIO Identify(%+v)", msg)
		return
	}

	f := segmentClient.Identify
	if analyticsOptions(opts).Has(Synchronous) {
		if err := f(msg); err != nil {
			golog.Errorf("SegmentIO Identify(%+v) failed: %s", msg, err)
		}
	} else {
		conc.Go(func() {
			if err := f(msg); err != nil {
				golog.Errorf("SegmentIO Identify(%+v) failed: %s", msg, err)
			}
		})
	}

}

// SegmentPage calls the Page API using a segment client
// that lets you record whenever a user sees a page of your website,
// along with any properties about the page.
// https://segment.com/docs/spec/page/
func SegmentPage(msg *analytics.Page, opts ...AnalyticsOption) {
	if segmentClient == nil {
		golog.Infof("SegmentIO Page(%+v)", msg)
		return
	}

	f := segmentClient.Page
	if analyticsOptions(opts).Has(Synchronous) {
		if err := f(msg); err != nil {
			golog.Errorf("SegmentIO Page(%+v) failed: %s", msg, err)
		}
	} else {
		conc.Go(func() {
			if err := f(msg); err != nil {
				golog.Errorf("SegmentIO Page(%+v) failed: %s", msg, err)
			}
		})
	}
}

// SegmentTrack calls the Track API using a segment client
// that lets you record any actions a user performs
// https://segment.com/docs/spec/track/
func SegmentTrack(msg *analytics.Track, opts ...AnalyticsOption) {
	if segmentClient == nil {
		golog.Infof("SegmentIO Track(%+v)", msg)
		return
	}
	f := segmentClient.Track
	if analyticsOptions(opts).Has(Synchronous) {
		if err := f(msg); err != nil {
			golog.Errorf("SegmentIO Track(%+v) failed: %s", msg, err)
		}
	} else {
		conc.Go(func() {
			if err := f(msg); err != nil {
				golog.Errorf("SegmentIO Track(%+v) failed: %s", msg, err)
			}
		})
	}
}
