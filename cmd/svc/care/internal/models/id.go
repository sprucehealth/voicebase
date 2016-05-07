package models

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	modellib "github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/svc/care"
)

type VisitID struct {
	modellib.ObjectID
}

func NewVisitID() (VisitID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return VisitID{}, errors.Trace(err)
	}

	return VisitID{
		modellib.ObjectID{
			Prefix:  care.VisitIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

func ParseVisitID(s string) (VisitID, error) {
	id := EmptyVisitID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

func EmptyVisitID() VisitID {
	return VisitID{
		modellib.ObjectID{
			Prefix:  care.VisitIDPrefix,
			IsValid: false,
		},
	}
}
