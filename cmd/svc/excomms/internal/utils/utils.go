package utils

import (
	"sort"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/libs/errors"
)

func PersistRawMessage(dl dal.DAL, media map[string]*models.Media, msg *rawmsg.Incoming) (uint64, error) {
	var rawMessageID uint64
	if err := dl.Transact(func(d dal.DAL) error {
		mediaItems := make([]*models.Media, 0, len(media))
		for _, m := range media {
			mediaItems = append(mediaItems, m)
		}

		sort.Sort(models.ByMediaID(mediaItems))

		if err := d.StoreMedia(mediaItems); err != nil {
			return err
		}

		var err error
		rawMessageID, err = d.StoreIncomingRawMessage(msg)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return 0, errors.Trace(err)
	}

	return rawMessageID, nil
}
