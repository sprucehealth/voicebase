package admin

import (
	"errors"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/context"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/financial"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type incomingFinancialItemsAPIHandler struct {
	financialAccess financial.Financial
}

type outgoingFinancialItemsAPIHandler struct {
	financialAccess financial.Financial
}

type financialTransactionsRequest struct {
	From time.Time
	To   time.Time
}

type incomingFinancialItemsResponse struct {
	Items []*financial.IncomingItem `json:"items"`
}

type outgoingFinancialItemsResponse struct {
	Items []*financial.OutgoingItem `json:"items"`
}

const (
	durationSixMonths = 6 * 30 * 24 * time.Hour
)

var (
	dateSeperators = []rune{'-', '/'}
)

func NewIncomingFinancialItemsHandler(financialAccess financial.Financial) http.Handler {
	return httputil.SupportedMethods(&incomingFinancialItemsAPIHandler{
		financialAccess: financialAccess,
	}, httputil.Get)
}

func NewOutgoingFinancialItemsHandler(financialAccess financial.Financial) http.Handler {
	return httputil.SupportedMethods(&outgoingFinancialItemsAPIHandler{
		financialAccess: financialAccess,
	}, httputil.Get)
}

func parseRequest(r *http.Request) (*financialTransactionsRequest, error) {
	from := r.FormValue("from")
	if from == "" {
		return nil, errors.New("missing 'from' time value in query")
	}
	to := r.FormValue("to")
	if to == "" {
		return nil, errors.New("missing 'to' time value in query")
	}

	fromTime, err := encoding.ParseDate(from, "YMD", dateSeperators, 0)
	if err != nil {
		fromTime, err = encoding.ParseDate(from, "MDY", dateSeperators, 0)
		if err != nil {
			return nil, err
		}
	}

	toTime, err := encoding.ParseDate(to, "YMD", dateSeperators, 0)
	if err != nil {
		toTime, err = encoding.ParseDate(to, "MDY", dateSeperators, 0)
		if err != nil {
			return nil, err
		}
	}

	if toTime.ToTime().Before(fromTime.ToTime()) {
		return nil, errors.New("'to' time cannot be before 'from' time in query")
	}

	if toTime.ToTime().Sub(fromTime.ToTime()) > durationSixMonths {
		return nil, errors.New("Cannot query for more than 6 months worth of data.")
	}

	return &financialTransactionsRequest{
		From: fromTime.ToTime(),
		To:   toTime.ToTime(),
	}, nil
}

func (f *incomingFinancialItemsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rd, err := parseRequest(r)
	if err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "IncomingFinancialItems", map[string]interface{}{
		"from": rd.From,
		"to":   rd.To,
	})

	items, err := f.financialAccess.IncomingItems(rd.From, rd.To)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &incomingFinancialItemsResponse{
		Items: items,
	})
}

func (f *outgoingFinancialItemsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rd, err := parseRequest(r)
	if err != nil {
		www.APIBadRequestError(w, r, err.Error())
		return
	}

	account := context.Get(r, www.CKAccount).(*common.Account)
	audit.LogAction(account.ID, "AdminAPI", "OutgoingFinancialItems", map[string]interface{}{
		"from": rd.From,
		"to":   rd.To,
	})

	items, err := f.financialAccess.OutgoingItems(rd.From, rd.To)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, &outgoingFinancialItemsResponse{
		Items: items,
	})
}
