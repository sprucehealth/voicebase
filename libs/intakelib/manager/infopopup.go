package manager

import (
	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type infoPopupImage struct {
	ImageLink   string  `json:"url"`
	AspectRatio float32 `json:"aspect_ratio"`
	Caption     string  `json:"caption"`
}

func (i *infoPopupImage) staticInfoCopy(context map[string]string) interface{} {
	return &infoPopupImage{
		ImageLink:   i.ImageLink,
		AspectRatio: i.AspectRatio,
		Caption:     i.Caption,
	}
}

type infoPopup struct {
	Text   string            `json:"text"`
	Images []*infoPopupImage `json:"images"`
}

func (i *infoPopup) staticInfoCopy(context map[string]string) interface{} {
	iCopy := &infoPopup{
		Text:   i.Text,
		Images: make([]*infoPopupImage, len(i.Images)),
	}

	for j, image := range i.Images {
		iCopy.Images[j] = image.staticInfoCopy(context).(*infoPopupImage)
	}

	return iCopy
}

func (i *infoPopup) transformToProtobuf() (proto.Message, error) {
	ip := &intake.InfoPopup{
		Text:   proto.String(i.Text),
		Images: make([]*intake.InfoPopup_InfoPopupImage, len(i.Images)),
	}

	for i, image := range i.Images {
		ip.Images[i] = &intake.InfoPopup_InfoPopupImage{
			ImageLink:   proto.String(image.ImageLink),
			AspectRatio: proto.Float32(image.AspectRatio),
			Caption:     proto.String(image.Caption),
		}
	}

	return ip, nil
}

func populatePopup(data dataMap) (*infoPopup, error) {
	if !data.exists("popup") {
		return nil, nil
	}

	popupMap, err := data.dataMapForKey("popup")
	if err != nil {
		return nil, err
	}

	popup := &infoPopup{}
	return popup, popup.unmarshalMapFromClient(popupMap)
}

func (i *infoPopup) unmarshalMapFromClient(data dataMap) error {
	i.Text = data.mustGetString("text")

	if !data.exists("images") {
		return nil
	}

	images, err := data.getInterfaceSlice("images")
	if err != nil {
		return err
	}

	i.Images = make([]*infoPopupImage, len(images))
	for j, imageVal := range images {
		imageMap, err := getDataMap(imageVal)
		if err != nil {
			return err
		}

		i.Images[j] = &infoPopupImage{}
		if err := i.Images[j].unmarshalMapFromClient(imageMap); err != nil {
			return nil
		}
	}

	return nil
}

func (i *infoPopupImage) unmarshalMapFromClient(data dataMap) error {
	i.ImageLink = data.mustGetString("url")
	i.Caption = data.mustGetString("caption")
	i.AspectRatio = data.mustGetFloat32("aspect_ratio")
	if i.ImageLink != "" && i.AspectRatio == 0 {
		i.AspectRatio = 1.5
	}
	return nil
}
