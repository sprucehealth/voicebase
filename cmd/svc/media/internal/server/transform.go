package server

import (
	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/mime"
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
	return &media.MediaInfo{
		ID:         m.ID.String(),
		URL:        media.URL(s.mediaAPIDomain, m.ID.String()),
		ThumbURL:   media.ThumbnailURL(s.mediaAPIDomain, m.ID.String(), 0, 0, false),
		OwnerID:    m.OwnerID,
		OwnerType:  media.MediaOwnerType(media.MediaOwnerType_value[m.OwnerType.String()]),
		SizeBytes:  m.SizeBytes,
		DurationNS: m.DurationNS,
		MIME: &media.MIME{
			Type:    t.Type,
			Subtype: t.Subtype,
		},
	}, nil
}
