package models

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	modellib "github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/svc/layout"
)

type VisitLayoutID struct {
	modellib.ObjectID
}

func NewVisitLayoutID() (VisitLayoutID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return VisitLayoutID{}, errors.Trace(err)
	}
	return VisitLayoutID{
		modellib.ObjectID{
			Prefix:  layout.VisitLayoutIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

func ParseVisitLayoutID(s string) (VisitLayoutID, error) {
	id := EmptyVisitLayoutID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

func EmptyVisitLayoutID() VisitLayoutID {
	return VisitLayoutID{
		modellib.ObjectID{
			Prefix:  layout.VisitLayoutIDPrefix,
			IsValid: false,
		},
	}
}

type VisitCategoryID struct {
	modellib.ObjectID
}

func NewVisitCategoryID() (VisitCategoryID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return VisitCategoryID{}, errors.Trace(err)
	}
	return VisitCategoryID{
		modellib.ObjectID{
			Prefix:  layout.VisitCategoryIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

func ParseVisitCategoryID(s string) (VisitCategoryID, error) {
	id := EmptyVisitCategoryID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

func EmptyVisitCategoryID() VisitCategoryID {
	return VisitCategoryID{
		modellib.ObjectID{
			Prefix:  layout.VisitCategoryIDPrefix,
			IsValid: false,
		},
	}
}

type VisitLayoutVersionID struct {
	modellib.ObjectID
}

func NewVisitLayoutVersionID() (VisitLayoutVersionID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return VisitLayoutVersionID{}, errors.Trace(err)
	}
	return VisitLayoutVersionID{
		modellib.ObjectID{
			Prefix:  layout.VisitLayoutVersionIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

func ParseVisitLayoutVersionID(s string) (VisitLayoutVersionID, error) {
	id := EmptyVisitLayoutVersionID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

func EmptyVisitLayoutVersionID() VisitLayoutVersionID {
	return VisitLayoutVersionID{
		modellib.ObjectID{
			Prefix:  layout.VisitLayoutVersionIDPrefix,
			IsValid: false,
		},
	}
}
