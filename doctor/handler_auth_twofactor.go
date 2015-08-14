package doctor

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/auth"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"golang.org/x/net/context"
)

type twoFactorHandler struct {
	authAPI             api.AuthAPI
	dataAPI             api.DataAPI
	apiDomain           string
	smsAPI              api.SMSAPI
	fromNumber          string
	twoFactorExpiration int
}

type TwoFactorRequest struct {
	TwoFactorToken string `json:"two_factor_token"`
	Code           string `json:"code"`
	Resend         bool   `json:"bool"`
}

func NewTwoFactorHandler(
	dataAPI api.DataAPI,
	authAPI api.AuthAPI,
	smsAPI api.SMSAPI,
	apiDomain,
	fromNumber string,
	twoFactorExpiration int,
) httputil.ContextHandler {
	return apiservice.RequestCacheHandler(
		apiservice.AuthorizationRequired(&twoFactorHandler{
			dataAPI:             dataAPI,
			authAPI:             authAPI,
			smsAPI:              smsAPI,
			apiDomain:           apiDomain,
			fromNumber:          fromNumber,
			twoFactorExpiration: twoFactorExpiration,
		}))
}

func (d *twoFactorHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	if r.Method != httputil.Post {
		return false, apiservice.NewResourceNotFoundError("", r)
	}
	var req TwoFactorRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}
	account, err := d.authAPI.ValidateTempToken(api.TwoFactorAuthToken, req.TwoFactorToken)
	if err == api.ErrTokenDoesNotExist || err == api.ErrTokenExpired {
		return false, apiservice.NewAuthTimeoutError()
	} else if err != nil {
		return false, err
	}
	requestCache := apiservice.MustCtxCache(ctx)
	requestCache[apiservice.CKAccount] = account
	requestCache[apiservice.CKRequestData] = &req
	return true, nil
}

func (d *twoFactorHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := requestCache[apiservice.CKAccount].(*common.Account)
	req := requestCache[apiservice.CKRequestData].(*TwoFactorRequest)

	appHeaders := apiservice.ExtractSpruceHeaders(r)

	if req.Resend {
		if _, err := auth.SendTwoFactorCode(d.authAPI, d.smsAPI, d.fromNumber, account.ID, appHeaders.DeviceID, d.twoFactorExpiration); err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}
		apiservice.WriteJSONSuccess(w)
		return
	}

	codeToken := auth.TwoFactorCodeToken(account.ID, appHeaders.DeviceID, req.Code)

	account, err := d.authAPI.ValidateTempToken(api.TwoFactorAuthCode, codeToken)
	if err == api.ErrTokenDoesNotExist {
		apiservice.WriteUserError(w, http.StatusForbidden, "Invalid verification code")
		return
	} else if err == api.ErrTokenExpired {
		apiservice.WriteError(ctx, apiservice.NewAccessForbiddenError(), w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// Mark this device as being verified with two factor
	if err := d.authAPI.UpdateAccountDeviceVerification(account.ID, appHeaders.DeviceID, true); err != nil {
		// Don't return an error since the person can still continue even if this fails for whatever reason
		golog.Errorf(err.Error())
	}

	token, err := d.authAPI.CreateToken(account.ID, api.Mobile, 0)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	doctor, err := d.dataAPI.GetDoctorFromAccountID(account.ID)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	go func() {
		if err := d.authAPI.DeleteTempToken(api.TwoFactorAuthCode, codeToken); err != nil {
			golog.Errorf(err.Error())
		}
	}()

	httputil.JSONResponse(w, http.StatusOK, &AuthenticationResponse{
		Token:  token,
		Doctor: responses.TransformDoctor(doctor, d.apiDomain),
	})
}
