package main

import (
	"github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/libs/golog"
)

type segmentIOWrapper struct {
	*analytics.Client
}

func (sio *segmentIOWrapper) Alias(msg *analytics.Alias) error {
	if sio.Client == nil {
		golog.Infof("SegmentIO Alias(%+v)", msg)
		return nil
	}
	go func() {
		if err := sio.Client.Alias(msg); err != nil {
			golog.Errorf("SegmentIO Alias(%+v) failed: %s", msg, err)
		}
	}()
	return nil
}

func (sio *segmentIOWrapper) Group(msg *analytics.Group) error {
	if sio.Client == nil {
		golog.Infof("SegmentIO Group(%+v)", msg)
		return nil
	}
	go func() {
		if err := sio.Client.Group(msg); err != nil {
			golog.Errorf("SegmentIO Group(%+v) failed: %s", msg, err)
		}
	}()
	return nil
}

func (sio *segmentIOWrapper) Identify(msg *analytics.Identify) error {
	if sio.Client == nil {
		golog.Infof("SegmentIO Identify(%+v)", msg)
		return nil
	}
	go func() {
		if err := sio.Client.Identify(msg); err != nil {
			golog.Errorf("SegmentIO Identify(%+v) failed: %s", msg, err)
		}
	}()
	return nil
}

func (sio *segmentIOWrapper) Page(msg *analytics.Page) error {
	if sio.Client == nil {
		golog.Infof("SegmentIO Page(%+v)", msg)
		return nil
	}
	go func() {
		if err := sio.Client.Page(msg); err != nil {
			golog.Errorf("SegmentIO Page(%+v) failed: %s", msg, err)
		}
	}()
	return nil
}

func (sio *segmentIOWrapper) Track(msg *analytics.Track) error {
	if sio.Client == nil {
		golog.Infof("SegmentIO Track(%+v)", msg)
		return nil
	}
	go func() {
		if err := sio.Client.Track(msg); err != nil {
			golog.Errorf("SegmentIO Track(%+v) failed: %s", msg, err)
		}
	}()
	return nil
}
