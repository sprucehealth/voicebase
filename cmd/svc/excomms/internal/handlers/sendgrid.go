package handlers

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg/sendgrid"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/utils"
	"github.com/sprucehealth/backend/libs/conc"
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
	} else if sgi.Headers == "" || sgi.SMTPEnvelope == "" {
		golog.Warningf("Unable to parse the sendgrid parameters from the request")
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

	rawMessageID, err := utils.PersistRawMessage(e.dal, media, msg)
	if err != nil {
		golog.Errorf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// publish the stored message to SNS
	conc.Go(func() {
		if err := sns.Publish(e.snsCLI, e.snsTopic, &sns.IncomingRawMessageNotification{
			ID: rawMessageID,
		}); err != nil {
			golog.Errorf(err.Error())
		}
	})
}
