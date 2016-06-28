package manager

import (
	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type genericPopupScreen struct {
	*screenInfo
	ViewDataJSON []byte `json:"views"`
}

func (g *genericPopupScreen) staticInfoCopy(context map[string]string) interface{} {
	gCopy := &genericPopupScreen{
		screenInfo:   g.screenInfo.staticInfoCopy(context).(*screenInfo),
		ViewDataJSON: make([]byte, len(g.ViewDataJSON)),
	}

	copy(gCopy.ViewDataJSON, g.ViewDataJSON)

	return gCopy
}

func (g *genericPopupScreen) unmarshalMapFromClient(data dataMap, parent layoutUnit, dataSource questionAnswerDataSource) error {
	if err := data.requiredKeys(screenTypeGenericPopup.String(), "client_data"); err != nil {
		return err
	}

	var err error
	g.screenInfo, err = populateScreenInfo(data, parent)
	if err != nil {
		return err
	}

	clientData, err := data.dataMapForKey("client_data")
	if err != nil {
		return err
	}

	g.ViewDataJSON, err = clientData.getJSONData("views")
	if err != nil {
		return err
	}

	return nil
}

func (g *genericPopupScreen) TypeName() string {
	return screenTypeGenericPopup.String()
}

func (g *genericPopupScreen) transformToProtobuf() (proto.Message, error) {
	sInfo, err := transformScreenInfoToProtobuf(g.screenInfo)
	if err != nil {
		return nil, err
	}

	return &intake.GenericPopupScreen{
		ScreenInfo:   sInfo.(*intake.CommonScreenInfo),
		ViewDataJson: g.ViewDataJSON,
	}, nil
}
