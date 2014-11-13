package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/www"
)

type stripeChargesAPIHandler struct {
	st *stripe.StripeService
}

// Stripped down version of a stripe charge
type stripeCharge struct {
	Created  time.Time `json:"created"`
	Livemode bool      `json:"livemode"`
	Paid     bool      `json:"paid"`
	Amount   int       `json:"amount"`
	Currency string    `json:"currency"`
	Refunded bool      `json:"refunded"`
}

func NewStripeChargesAPIHAndler(st *stripe.StripeService) http.Handler {
	return httputil.SupportedMethods(&stripeChargesAPIHandler{
		st: st,
	}, []string{"GET"})
}

func (h *stripeChargesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.st == nil {
		www.JSONResponse(w, r, http.StatusOK, nil)
		return
	}

	limit := 10
	if s := r.FormValue("limit"); s != "" {
		var err error
		limit, err = strconv.Atoi(s)
		if err != nil {
			www.APIBadRequestError(w, r, "limit must be an integer")
		}
	}

	charges, err := h.st.ListAllCharges(limit)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	res := make([]*stripeCharge, len(charges))
	for i, c := range charges {
		res[i] = &stripeCharge{
			Created:  c.Created.UTC(),
			Livemode: c.Livemode,
			Paid:     c.Paid,
			Amount:   c.Amount,
			Currency: c.Currency,
			Refunded: c.Refunded,
		}
	}

	www.JSONResponse(w, r, http.StatusOK, res)
}
