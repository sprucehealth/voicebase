package manager

import (
	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type mediaIDReplacement struct {
	ID           string
	URL          string
	ThumbnailURL string
}

func (p *mediaIDReplacement) unmarshalProtobuf(data []byte) error {
	var pir intake.MediaIDReplacement
	if err := proto.Unmarshal(data, &pir); err != nil {
		return err
	}

	p.ID = *pir.Id
	if pir.Url != nil {
		p.URL = *pir.Url
	}
	if pir.ThumbnailUrl != nil {
		p.ThumbnailURL = *pir.ThumbnailUrl
	}
	return nil
}
