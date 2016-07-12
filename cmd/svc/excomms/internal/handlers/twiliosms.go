package handlers

import (
	"net/http"
	"time"

	"context"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/sns"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/twilio"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/twilio/twiml"
)

type twilioSMSHandler struct {
	dal      dal.DAL
	snsTopic string
	snsCLI   snsiface.SNSAPI
}

func NewTwilioSMSHandler(dal dal.DAL, snsTopic string, snsCLI snsiface.SNSAPI) httputil.ContextHandler {
	return &twilioSMSHandler{
		dal:      dal,
		snsTopic: snsTopic,
		snsCLI:   snsCLI,
	}
}

func (t *twilioSMSHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	tw, err := twilio.ParamsFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rm := &rawmsg.Incoming{
		Type: rawmsg.Incoming_TWILIO_SMS,
		Message: &rawmsg.Incoming_Twilio{
			Twilio: tw,
		},
		Timestamp: uint64(time.Now().Unix()),
	}

	// store in database
	rawMessageID, err := t.dal.StoreIncomingRawMessage(rm)
	if err != nil {
		golog.Errorf(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// publish to sns
	conc.Go(func() {
		if err := sns.Publish(t.snsCLI, t.snsTopic, &sns.IncomingRawMessageNotification{
			ID: rawMessageID,
		}); err != nil {
			golog.Errorf(err.Error())
		}
	})

	res := twiml.Response{}
	w.WriteHeader(http.StatusOK)
	res.WriteResponse(w)
}
