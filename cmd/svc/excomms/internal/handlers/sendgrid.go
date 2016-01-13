package handlers

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sendgrid"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"

	"golang.org/x/net/context"
)

type sendgridHandler struct {
	snsTopic string
	snsCLI   snsiface.SNSAPI
	dal      dal.DAL
}

func NewSendGridHandler(snsTopic string, snsCLI snsiface.SNSAPI, dal dal.DAL) httputil.ContextHandler {
	return &sendgridHandler{
		snsTopic: snsTopic,
		snsCLI:   snsCLI,
		dal:      dal,
	}
}

func (e *sendgridHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	sgi, err := sendgrid.ParamsFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO: Upload any media attachments to S3

	msg := &rawmsg.Incoming{
		Type: rawmsg.Incoming_SENDGRID_EMAIL,
		Message: &rawmsg.Incoming_SendGrid{
			SendGrid: sgi,
		},
		Timestamp: uint64(time.Now().Unix()),
	}

	// persist the message to the database
	rawMessageID, err := e.dal.StoreIncomingRawMessage(msg)
	if err != nil {
		golog.Errorf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// publish the stored message to SNS
	sns.Publish(e.snsCLI, e.snsTopic, &sns.IncomingRawMessageNotification{
		ID: rawMessageID,
	})

}
