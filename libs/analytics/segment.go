package analytics

import (
	"github.com/segmentio/analytics-go"
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

// SegmentAlias calls the Alias API using a segment client to
// merge two user identities.
// https://segment.com/docs/spec/alias/
func SegmentAlias(msg *analytics.Alias) {
	if segmentClient == nil {
		golog.Infof("SegmentIO Alias(%+v)", msg)
		return
	}
	conc.Go(func() {
		if err := segmentClient.Alias(msg); err != nil {
			golog.Errorf("SegmentIO Alias(%+v) failed: %s", msg, err)
		}
	})
}

// SegmentGroup calls the Group API using a segment client
// to associate an individual user with a group.
// https://segment.com/docs/spec/group/
func SegmentGroup(msg *analytics.Group) {
	if segmentClient == nil {
		golog.Infof("SegmentIO Group(%+v)", msg)
		return
	}
	conc.Go(func() {
		if err := segmentClient.Group(msg); err != nil {
			golog.Errorf("SegmentIO Group(%+v) failed: %s", msg, err)
		}
	})
}

// SegmentIdentify calls the Identify API using a segment client
// that ties a customer and their actions to a recognizable ID
// https://segment.com/docs/spec/identify/
func SegmentIdentify(msg *analytics.Identify) {
	if segmentClient == nil {
		golog.Infof("SegmentIO Identify(%+v)", msg)
		return
	}
	conc.Go(func() {
		if err := segmentClient.Identify(msg); err != nil {
			golog.Errorf("SegmentIO Identify(%+v) failed: %s", msg, err)
		}
	})
}

// SegmentPage calls the Page API using a segment client
// that lets you record whenever a user sees a page of your website,
// along with any properties about the page.
// https://segment.com/docs/spec/page/
func SegmentPage(msg *analytics.Page) {
	if segmentClient == nil {
		golog.Infof("SegmentIO Page(%+v)", msg)
		return
	}
	conc.Go(func() {
		if err := segmentClient.Page(msg); err != nil {
			golog.Errorf("SegmentIO Page(%+v) failed: %s", msg, err)
		}
	})
}

// SegmentTrack calls the Track API using a segment client
// that lets you record any actions a user performs
// https://segment.com/docs/spec/track/
func SegmentTrack(msg *analytics.Track) {
	if segmentClient == nil {
		golog.Infof("SegmentIO Track(%+v)", msg)
		return
	}
	conc.Go(func() {
		if err := segmentClient.Track(msg); err != nil {
			golog.Errorf("SegmentIO Track(%+v) failed: %s", msg, err)
		}
	})
}
