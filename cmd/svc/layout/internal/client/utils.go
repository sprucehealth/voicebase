package client

import (
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
)

func transformBody(body *saml.ScreenBody) *layout.Body {
	if body == nil {
		return nil
	}

	return &layout.Body{
		Text: body.Text,
	}
}

func transformPopup(popup *saml.Popup) *layout.Popup {
	if popup == nil {
		return nil
	}

	return &layout.Popup{
		Text: popup.Text,
	}
}
