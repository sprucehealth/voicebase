package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type replaceCardHandler struct {
	dataAPI    api.DataAPI
	paymentAPI apiservice.StripeClient
}

func NewReplaceCardHandler(
	dataAPI api.DataAPI,
	paymentAPI apiservice.StripeClient) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&replaceCardHandler{
					dataAPI:    dataAPI,
					paymentAPI: paymentAPI,
				}), api.RolePatient), httputil.Put)
}

type replaceCardRequestData struct {
	Card *common.Card `json:"card"`
}

type replaceCardResponse struct {
	Cards []*common.Card
}

func (p *replaceCardHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var rd replaceCardRequestData
	if err := apiservice.DecodeRequestData(&rd, r); err != nil {
		apiservice.WriteValidationError(ctx, err.Error(), w, r)
		return
	}

	if rd.Card == nil {
		apiservice.WriteValidationError(ctx, "Card not specified", w, r)
		return
	} else if rd.Card.Token == "" {
		apiservice.WriteValidationError(ctx, "unique card token not specified", w, r)
		return
	}
	// make the card being added a default
	rd.Card.IsDefault = true

	// check if the patient has any existing card to be deleted
	patient, err := p.dataAPI.GetPatientFromAccountID(apiservice.MustCtxAccount(ctx).ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	cards, err := p.dataAPI.GetCardsForPatient(patient.ID.Int64())
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// first add the new card
	enforceAddressRequirement := false
	if err := addCardForPatient(
		p.dataAPI,
		p.paymentAPI,
		nil,
		rd.Card,
		patient,
		enforceAddressRequirement); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// delete old card if it exists
	if len(cards) > 0 {
		switchDefaultCard := false
		if err := deleteCard(
			cards[0].ID.Int64(),
			patient,
			switchDefaultCard,
			p.dataAPI, p.paymentAPI); err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
	}

	apiservice.WriteJSONSuccess(w)
}
