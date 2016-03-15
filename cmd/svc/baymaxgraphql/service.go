package main

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	lmedia "github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type service struct {
	notification    notification.Client
	settings        settings.SettingsClient
	invite          invite.InviteClient
	mediaSigner     *media.Signer
	emailDomain     string
	webDomain       string
	staticURLPrefix string
	spruceOrgID     string
	segmentio       *segmentIOWrapper
	media           *lmedia.Service
	// TODO: Remove this
	serviceNumber phone.Number
}

func hydrateThreads(ctx context.Context, ram raccess.ResourceAccessor, threads []*models.Thread) error {
	if len(threads) == 0 {
		return nil
	}

	// TODO: for now requiring that all threads are in the same org which is currently the case
	orgID := threads[0].OrganizationID
	for _, t := range threads[1:] {
		if t.OrganizationID != orgID {
			return errors.Trace(fmt.Errorf("org %s doesn't match %s", t.OrganizationID, orgID))
		}
	}

	// Get the set of entitiy IDs we need to lookup
	var entityIDs []string
	for _, t := range threads {
		if t.PrimaryEntity == nil && t.PrimaryEntityID != "" {
			entityIDs = append(entityIDs, t.PrimaryEntityID)
		}
	}

	var eMap map[string]*directory.Entity
	if len(entityIDs) != 0 {
		entities, err := ram.Entities(ctx, orgID, dedupeStrings(entityIDs), []directory.EntityInformation{directory.EntityInformation_CONTACTS})
		if err != nil {
			return errors.Trace(err)
		}
		eMap = make(map[string]*directory.Entity, len(entities))
		for _, e := range entities {
			eMap[e.ID] = e
		}
	}

	for _, t := range threads {
		if t.MessageCount == 0 && t.Type == models.ThreadTypeTeam {
			t.EmptyStateTextMarkup = "This is the beginning of a conversation that is visible to everyone in your organization.\n\nInvite some colleagues to join and then send a message here to get things started."
		}
		if t.PrimaryEntityID != "" && (t.Type == models.ThreadTypeExternal || t.Title == "") {
			if t.PrimaryEntity == nil {
				t.PrimaryEntity = eMap[t.PrimaryEntityID]
				if t.PrimaryEntity == nil {
					return errors.Trace(fmt.Errorf("primary entity %s not found for thread %s", t.PrimaryEntityID, t.ID))
				}
			}
			t.Title = threadTitleForEntity(t.PrimaryEntity)
			// TODO: remove this once old threads are migrated
			if t.Type == models.ThreadTypeUnknown {
				// TODO: checking the thread title is crazy brittle but for now don't have a way to tell apart SYSTEM entities
				t.AllowInternalMessages = t.PrimaryEntity.Type == directory.EntityType_EXTERNAL || (t.PrimaryEntity.Type == directory.EntityType_SYSTEM && !strings.HasPrefix(t.Title, "Team "))
				t.AllowDelete = t.PrimaryEntity.Type == directory.EntityType_EXTERNAL
			}
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
			Channel: excomms.ChannelType_SMS,
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

func (s *service) inviteInfo(ctx context.Context) (*invite.LookupInviteResponse, error) {
	sh := gqlctx.SpruceHeaders(ctx)
	if sh == nil || sh.DeviceID == "" {
		return nil, nil
	}

	res, err := s.invite.AttributionData(ctx, &invite.AttributionDataRequest{
		DeviceID: sh.DeviceID,
	})
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, errors.Trace(err)
	}

	var inviteToken string
	for _, v := range res.Values {
		if v.Key == "invite_token" {
			inviteToken = v.Value
			break
		}
	}
	if inviteToken == "" {
		return nil, nil
	}

	ires, err := s.invite.LookupInvite(ctx, &invite.LookupInviteRequest{
		Token: inviteToken,
	})
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, errors.Trace(err)
	}

	return ires, nil
}
