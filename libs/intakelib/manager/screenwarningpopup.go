package manager

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type warningPopupScreen struct {
	*screenInfo
	ContentHeaderTitle string  `json:"content_header_title"`
	BottomButtonTitle  string  `json:"bottom_button_title"`
	Body               *body   `json:"body"`
	ImageWidth         float32 `json:"image_width"`
	ImageHeight        float32 `json:"image_height"`
	ImageLink          string  `json:"image_link"`
}

func (w *warningPopupScreen) staticInfoCopy(context map[string]string) interface{} {
	wCopy := &warningPopupScreen{
		screenInfo:         w.screenInfo.staticInfoCopy(context).(*screenInfo),
		ContentHeaderTitle: w.ContentHeaderTitle,
		BottomButtonTitle:  w.BottomButtonTitle,
		ImageWidth:         w.ImageWidth,
		ImageHeight:        w.ImageHeight,
		ImageLink:          w.ImageLink,
	}

	if w.Body != nil {
		wCopy.Body = w.Body.staticInfoCopy(context).(*body)
	}

	return wCopy
}

func (w *warningPopupScreen) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	if err := data.requiredKeys(screenTypeWarningPopup.String(), "body"); err != nil {
		return err
	}

	var err error
	w.screenInfo, err = populateScreenInfo(data, parent)
	if err != nil {
		return err
	}

	w.BottomButtonTitle = data.mustGetString("bottom_button_title")
	w.ContentHeaderTitle = data.mustGetString("content_header_title")
	w.ImageWidth = 100
	w.ImageHeight = 100
	w.ImageLink = "spruce:///image/icon_triage_alert"
	w.Body, err = populateBody(data)
	if err != nil {
		return err
	}

	return nil
}

func (w *warningPopupScreen) TypeName() string {
	return screenTypeWarningPopup.String()
}

func (w *warningPopupScreen) transformToProtobuf() (proto.Message, error) {
	sInfo, err := transformScreenInfoToProtobuf(w.screenInfo)
	if err != nil {
		return nil, err
	}

	var transformedBody *intake.Body
	if w.Body != nil {
		body, err := w.Body.transformToProtobuf()
		if err != nil {
			return nil, err
		}
		transformedBody = body.(*intake.Body)
	}

	return &intake.ImagePopupScreen{
		ScreenInfo:         sInfo.(*intake.CommonScreenInfo),
		Body:               transformedBody,
		ImageWidth:         proto.Float32(w.ImageWidth),
		ImageHeight:        proto.Float32(w.ImageHeight),
		ImageLink:          proto.String(w.ImageLink),
		ContentHeaderTitle: proto.String(w.ContentHeaderTitle),
		BottomButtonTitle:  proto.String(w.BottomButtonTitle),
	}, nil
}

func (w *warningPopupScreen) String() string {
	return fmt.Sprintf("-- %s: %s | %s", w.layoutUnitID(), w.TypeName(), w.v)
}
