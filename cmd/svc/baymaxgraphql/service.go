package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type service struct {
	notification             notification.Client
	settings                 settings.SettingsClient
	invite                   invite.InviteClient
	layout                   layout.LayoutClient
	care                     care.CareClient
	emailDomain              string
	webDomain                string
	mediaAPIDomain           string
	inviteAPIDomain          string
	staticURLPrefix          string
	spruceOrgID              string
	sns                      snsiface.SNSAPI
	supportMessageTopicARN   string
	emailTemplateIDs         emailTemplateIDs
	transactionalEmailSender string
	stripeConnectURL         string
	hintConnectURL           string
	intercomSecretKey        string
	publisher                events.Publisher
	// TODO: Remove this
	serviceNumber phone.Number
	layoutStore   layout.Storage
}

type emailTemplateIDs struct {
	passwordReset     string
	emailVerification string
}

func hydrateThreads(ctx context.Context, ram raccess.ResourceAccessor, threads []*models.Thread) error {
	if len(threads) == 0 {
		return nil
	}
	// TODO: for now requiring that all threads are in the same org which is currently the case
	orgID := threads[0].OrganizationID
	for _, t := range threads[1:] {
		if t.OrganizationID != orgID {
			return errors.Errorf("org %s doesn't match %s", t.OrganizationID, orgID)
		}
	}
	return nil
}

// createAndSendSMSVerificationCode creates a verification code and asynchronously sends it via
// SMS to the provided number. The token associated with the code is returned. The phone number
// is expected to already be E164 format.
func createAndSendSMSVerificationCode(ctx context.Context, ram raccess.ResourceAccessor, serviceNumber phone.Number, codeType auth.VerificationCodeType, valueToVerify string, pn phone.Number) (string, error) {
	golog.Debugf("Creating and sending verification code of type %s to %s", auth.VerificationCodeType_name[int32(codeType)], pn)

	resp, err := ram.CreateVerificationCode(ctx, codeType, valueToVerify)
	if err != nil {
		return "", err
	}

	golog.Debugf("Sending code %s to %s for verification", resp.VerificationCode.Code, pn)
	conc.Go(func() {
		if err := ram.SendMessage(context.TODO(), &excomms.SendMessageRequest{
			DeprecatedChannel: excomms.ChannelType_SMS,
			Message: &excomms.SendMessageRequest_SMS{
				SMS: &excomms.SMSMessage{
					Text:            fmt.Sprintf("Your Spruce verification code is %s", resp.VerificationCode.Code),
					FromPhoneNumber: serviceNumber.String(),
					ToPhoneNumber:   pn.String(),
				},
			},
		}); err != nil {
			golog.Errorf("Error while sending phone number verification message for %s: %s", pn, err)
		}
	})
	return resp.VerificationCode.Token, nil
}

const inviteTokenAttributionKey = "invite_token"

func (s *service) inviteAndAttributionInfo(ctx context.Context, accountInviteClientID string) (*invite.LookupInviteResponse, map[string]string, error) {
	sh := devicectx.SpruceHeaders(ctx)
	if sh == nil || sh.DeviceID == "" {
		return nil, nil, nil
	}

	res, err := s.invite.AttributionData(ctx, &invite.AttributionDataRequest{
		DeviceID: sh.DeviceID + accountInviteClientID,
	})
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, nil, nil
		}
		return nil, nil, errors.Trace(err)
	}

	attribValues := make(map[string]string, len(res.Values))
	var inviteToken string
	for _, v := range res.Values {
		if v.Key == inviteTokenAttributionKey {
			inviteToken = v.Value
		}
		attribValues[v.Key] = v.Value
	}
	if inviteToken == "" {
		return nil, attribValues, nil
	}

	ires, err := s.invite.LookupInvite(ctx, &invite.LookupInviteRequest{
		InviteToken: inviteToken,
	})
	if grpc.Code(err) == codes.NotFound {
		return nil, attribValues, nil
	} else if err != nil {
		return nil, nil, errors.Trace(err)
	}

	return ires, attribValues, nil
}
