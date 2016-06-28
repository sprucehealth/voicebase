package manager

import (
	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type triageScreen struct {
	*screenInfo
	ContentHeaderTitle string `json:"content_header_title"`
	ContentButtonTitle string `json:"content_button_title"`
	BottomButtonTitle  string `json:"bottom_button_title"`
	Body               *body  `json:"body"`
}

func (t *triageScreen) staticInfoCopy(context map[string]string) interface{} {
	tCopy := &triageScreen{
		screenInfo:         t.screenInfo.staticInfoCopy(context).(*screenInfo),
		ContentHeaderTitle: t.ContentHeaderTitle,
		ContentButtonTitle: t.ContentButtonTitle,
		BottomButtonTitle:  t.BottomButtonTitle,
	}

	if t.Body != nil {
		tCopy.Body = t.Body.staticInfoCopy(context).(*body)
	}

	return tCopy
}

func (t *triageScreen) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	if err := data.requiredKeys(
		screenTypeTriage.String(),
		"content_header_title"); err != nil {
		return err
	}

	var err error
	t.screenInfo, err = populateScreenInfo(data, parent)
	if err != nil {
		return err
	}

	t.ContentHeaderTitle = data.mustGetString("content_header_title")
	t.ContentButtonTitle = data.mustGetString("content_button_title")
	t.BottomButtonTitle = data.mustGetString("bottom_button_title")
	t.Body, err = populateBody(data)
	if err != nil {
		return err
	}

	// default to true given that the type indicates this to be a triage screen
	t.screenInfo.IsTriageScreen = true

	return err
}

func (t *triageScreen) TypeName() string {
	return screenTypeTriage.String()
}

func (t *triageScreen) transformToProtobuf() (proto.Message, error) {
	sInfo, err := transformScreenInfoToProtobuf(t.screenInfo)
	if err != nil {
		return nil, err
	}

	body, err := t.Body.transformToProtobuf()
	if err != nil {
		return nil, err
	}

	return &intake.TriageScreen{
		ScreenInfo:         sInfo.(*intake.CommonScreenInfo),
		ContentHeaderTitle: proto.String(t.ContentHeaderTitle),
		BottomButtonTitle:  proto.String(t.BottomButtonTitle),
		Body:               body.(*intake.Body),
	}, nil
}
