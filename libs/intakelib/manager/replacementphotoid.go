package manager

import (
	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type photoIDReplacement struct {
	ID  string
	URL string
}

func (p *photoIDReplacement) unmarshalProtobuf(data []byte) error {
	var pir intake.PhotoIDReplacement
	if err := proto.Unmarshal(data, &pir); err != nil {
		return err
	}

	p.ID = *pir.Id
	if pir.Url != nil {
		p.URL = *pir.Url
	}
	return nil
}
