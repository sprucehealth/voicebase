package handlers

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sendgrid"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/storage"

	"golang.org/x/net/context"
)

type sendgridHandler struct {
	snsTopic string
	snsCLI   snsiface.SNSAPI
	dal      dal.DAL
	store    storage.Store
}

func NewSendGridHandler(snsTopic string, snsCLI snsiface.SNSAPI, dal dal.DAL, store storage.Store) httputil.ContextHandler {
	return &sendgridHandler{
		snsTopic: snsTopic,
		snsCLI:   snsCLI,
		dal:      dal,
		store:    store,
	}
}

func (e *sendgridHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	sgi, media, err := sendgrid.ParamsFromRequest(r, e.store)
	if err != nil {
		golog.Errorf(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	msg := &rawmsg.Incoming{
		Type: rawmsg.Incoming_SENDGRID_EMAIL,
		Message: &rawmsg.Incoming_SendGrid{
			SendGrid: sgi,
		},
		Timestamp: uint64(time.Now().Unix()),
	}

	var rawMessageID uint64
	if err := e.dal.Transact(func(dl dal.DAL) error {
		mediaItems := make([]*models.Media, 0, len(media))
		for _, m := range media {
			mediaItems = append(mediaItems, m)
		}

		if err := dl.StoreMedia(mediaItems); err != nil {
			return err
		}

		rawMessageID, err = dl.StoreIncomingRawMessage(msg)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		golog.Errorf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// publish the stored message to SNS
	sns.Publish(e.snsCLI, e.snsTopic, &sns.IncomingRawMessageNotification{
		ID: rawMessageID,
	})

}
