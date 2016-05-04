package client

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
)

type photoSlotID struct {
	model.ObjectID
}

func transformPhotoSlots(photoSlots []*saml.PhotoSlot) ([]*layout.PhotoSlot, error) {
	tPhotoSlots := make([]*layout.PhotoSlot, len(photoSlots))
	for i, photoSlot := range photoSlots {

		// SAML layer does not generate a tag (unique ID) for
		// photo slots so generate one here.
		id, err := idgen.NewID()
		if err != nil {
			return nil, errors.Trace(err)
		}

		slotID := &photoSlotID{
			model.ObjectID{
				Prefix:  "photoSlot_",
				Val:     id,
				IsValid: true,
			},
		}

		tPhotoSlots[i] = &layout.PhotoSlot{
			ID:       slotID.String(),
			Name:     photoSlot.Name,
			Required: photoSlot.Required,
		}

		tPhotoSlots[i].ClientData, err = transformPhotoSlotClientData(photoSlot.ClientData)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	return tPhotoSlots, nil
}

func transformPhotoSlotClientData(clientData *saml.PhotoSlotClientData) (*layout.PhotoSlotClientData, error) {
	if clientData == nil {
		return nil, nil
	}

	fs, err := layout.ParseFlashState(string(clientData.Flash))
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &layout.PhotoSlotClientData{
		PhotoTip: layout.PhotoTip{
			Tip:        clientData.Tip,
			TipStyle:   clientData.TipStyle,
			TipSubtext: clientData.TipSubtext,
		},
		OverlayImageURL:          clientData.OverlayImageURL,
		PhotoMissingErrorMessage: clientData.PhotoMissingErrorMessage,
		InitialCameraDirection:   clientData.InitialCameraDirection,
		Flash: fs,
		Tips:  transformPhotoTips(clientData.Tips),
	}, nil
}

func transformPhotoTips(tips map[string]*saml.PhotoTip) map[string]*layout.PhotoTip {
	tTips := make(map[string]*layout.PhotoTip)

	for key, tip := range tips {
		tTips[key] = &layout.PhotoTip{
			Tip:        tip.Tip,
			TipSubtext: tip.TipSubtext,
			TipStyle:   tip.TipStyle,
		}
	}

	return tTips
}
