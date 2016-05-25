package client

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
)

type mediaSlotID struct {
	model.ObjectID
}

func transformMediaSlots(mediaSlots []*saml.MediaSlot) ([]*layout.MediaSlot, error) {
	tMediaSlots := make([]*layout.MediaSlot, len(mediaSlots))
	for i, mediaSlot := range mediaSlots {

		// SAML layer does not generate a tag (unique ID) for
		// photo slots so generate one here.
		id, err := idgen.NewID()
		if err != nil {
			return nil, errors.Trace(err)
		}

		slotID := &mediaSlotID{
			model.ObjectID{
				Prefix:  "mediaSlot_",
				Val:     id,
				IsValid: true,
			},
		}

		tMediaSlots[i] = &layout.MediaSlot{
			ID:       slotID.String(),
			Name:     mediaSlot.Name,
			Required: mediaSlot.Required,
			Type:     mediaSlot.Type,
		}

		tMediaSlots[i].ClientData, err = transformMediaSlotClientData(mediaSlot.ClientData)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	return tMediaSlots, nil
}

func transformMediaSlotClientData(clientData *saml.MediaSlotClientData) (*layout.MediaSlotClientData, error) {
	if clientData == nil {
		return nil, nil
	}

	fs, err := layout.ParseFlashState(string(clientData.Flash))
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &layout.MediaSlotClientData{
		MediaTip: layout.MediaTip{
			Tip:        clientData.Tip,
			TipStyle:   clientData.TipStyle,
			TipSubtext: clientData.TipSubtext,
		},
		OverlayImageURL:          clientData.OverlayImageURL,
		PhotoMissingErrorMessage: clientData.PhotoMissingErrorMessage,
		MediaMissingErrorMessage: clientData.MediaMissingErrorMessage,
		InitialCameraDirection:   clientData.InitialCameraDirection,
		Flash: fs,
		Tips:  transformMediaTips(clientData.Tips),
	}, nil
}

func transformMediaTips(tips map[string]*saml.MediaTip) map[string]*layout.MediaTip {
	tTips := make(map[string]*layout.MediaTip)

	for key, tip := range tips {
		tTips[key] = &layout.MediaTip{
			Tip:        tip.Tip,
			TipSubtext: tip.TipSubtext,
			TipStyle:   tip.TipStyle,
		}
	}

	return tTips
}
