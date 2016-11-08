package server

import (
	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/mime"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/media"
)

func (s *server) transformMediasToResponse(ms []*dal.Media) ([]*media.MediaInfo, error) {
	rms := make([]*media.MediaInfo, len(ms))
	for i, m := range ms {
		rm, err := s.transformMediaToResponse(m)
		if err != nil {
			return nil, err
		}
		rms[i] = rm
	}
	return rms, nil
}

func (s *server) transformMediaToResponse(m *dal.Media) (*media.MediaInfo, error) {
	t, err := mime.ParseType(m.MimeType)
	if err != nil {
		return nil, err
	}
	ownerType, err := transformMediaOwnerTypeToResponse(m.OwnerType)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &media.MediaInfo{
		ID:         m.ID.String(),
		URL:        media.URL(s.mediaAPIDomain, m.ID.String(), m.MimeType),
		ThumbURL:   media.ThumbnailURL(s.mediaAPIDomain, m.ID.String(), m.MimeType, 0, 0, false),
		OwnerID:    m.OwnerID,
		OwnerType:  ownerType,
		SizeBytes:  m.SizeBytes,
		DurationNS: m.DurationNS,
		Name:       m.Name,
		MIME: &media.MIME{
			Type:    t.Type,
			Subtype: t.Subtype,
		},
		Public: m.Public,
	}, nil
}

func transformMediaOwnerTypeToResponse(ot dal.MediaOwnerType) (media.MediaOwnerType, error) {
	switch ot {
	case dal.MediaOwnerTypeOrganization:
		return media.MediaOwnerType_ORGANIZATION, nil
	case dal.MediaOwnerTypeThread:
		return media.MediaOwnerType_THREAD, nil
	case dal.MediaOwnerTypeEntity:
		return media.MediaOwnerType_ENTITY, nil
	case dal.MediaOwnerTypeAccount:
		return media.MediaOwnerType_ACCOUNT, nil
	case dal.MediaOwnerTypeVisit:
		return media.MediaOwnerType_VISIT, nil
	case dal.MediaOwnerTypeSavedMessage:
		return media.MediaOwnerType_SAVED_MESSAGE, nil
	case dal.MediaOwnerTypeLegacy:
		return media.MediaOwnerType_LEGACY, nil
	}
	return media.MediaOwnerType_OWNER_TYPE_UNKNOWN, errors.Errorf("unknown media owner type %s", ot)
}
