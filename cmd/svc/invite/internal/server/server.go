package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"

	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/invite/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/branch"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/invite/clientdata"
	"github.com/sprucehealth/backend/svc/settings"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var complexTokenGenerator common.TokenGenerator

const complexTokenLength = 16

type complexTokenGen struct{}

const phiAttributeText = "PROTECTED_PHI"

func (complexTokenGen) GenerateToken() (string, error) {
	b := make([]byte, complexTokenLength)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", errors.Trace(err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

var simpleTokenGenerator common.TokenGenerator

const simpleTokenLength = 6
const simpleTokenMaxValue = 999999

type simpleTokenGen struct{}

func (simpleTokenGen) GenerateToken() (string, error) {
	code, err := common.GenerateRandomNumber(simpleTokenMaxValue, simpleTokenLength)
	if err != nil {
		return "", errors.Trace(err)
	}
	return code, nil
}

func init() {
	simpleTokenGenerator = simpleTokenGen{}
	complexTokenGenerator = complexTokenGen{}
}

type server struct {
	dal                       dal.DAL
	clk                       clock.Clock
	directoryClient           directory.DirectoryClient
	excommsClient             excomms.ExCommsClient
	settingsClient            settings.SettingsClient
	branch                    branch.Client
	fromEmail                 string
	fromNumber                string
	eventsTopic               string
	sns                       snsiface.SNSAPI
	webInviteURL              *url.URL
	colleagueInviteTemplateID string
	patientInviteTemplateID   string
}

type popover struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	ButtonText string `json:"button_text"`
}

type organizationInvite struct {
	Popover popover `json:"popover"`
	OrgID   string  `json:"org_id"`
	OrgName string  `json:"org_name"`
}

type colleagueInviteClientData struct {
	OrganizationInvite organizationInvite `json:"organization_invite"`
}

type greeting struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	ButtonText string `json:"button_text"`
}

type patientInvite struct {
	Greeting greeting `json:"greeting"`
	OrgID    string   `json:"org_id"`
	OrgName  string   `json:"org_name"`
}

type patientInviteClientData struct {
	PatientInvite patientInvite `json:"patient_invite"`
}

type inviteDeliveryChannel string

const (
	inviteDeliverySMS   inviteDeliveryChannel = "SMS"
	inviteDeliveryEmail inviteDeliveryChannel = "EMAIL"
)

// New returns an initialized instance of the invite server
func New(
	dal dal.DAL,
	clk clock.Clock,
	directoryClient directory.DirectoryClient,
	excommsClient excomms.ExCommsClient,
	settingsClient settings.SettingsClient,
	snsC snsiface.SNSAPI,
	branch branch.Client,
	fromEmail, fromNumber, eventsTopic, webInviteURL string,
	colleagueInviteTemplateID, patientInviteTemplateID string,
) invite.InviteServer {
	if clk == nil {
		clk = clock.New()
	}
	var webURL *url.URL
	if webInviteURL != "" {
		var err error
		webURL, err = url.Parse(webInviteURL)
		if err != nil {
			golog.Fatalf("Failed to parse web invite URL: %s", err)
		}
	}
	return &server{
		dal:                       dal,
		clk:                       clk,
		directoryClient:           directoryClient,
		excommsClient:             excommsClient,
		settingsClient:            settingsClient,
		sns:                       snsC,
		branch:                    branch,
		fromEmail:                 fromEmail,
		fromNumber:                fromNumber,
		eventsTopic:               eventsTopic,
		webInviteURL:              webURL,
		colleagueInviteTemplateID: colleagueInviteTemplateID,
		patientInviteTemplateID:   patientInviteTemplateID,
	}
}

// AttributionData returns the attribution data for a device
func (s *server) AttributionData(ctx context.Context, in *invite.AttributionDataRequest) (*invite.AttributionDataResponse, error) {
	values, err := s.dal.AttributionData(ctx, in.DeviceID)
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, fmt.Sprintf("No attribution data found for device ID '%s'", in.DeviceID))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	out := &invite.AttributionDataResponse{
		Values: make([]*invite.AttributionValue, 0, len(values)),
	}
	for k, v := range values {
		out.Values = append(out.Values, &invite.AttributionValue{
			Key:   k,
			Value: v,
		})
	}
	return out, nil
}

// InviteColleagues sends invites to people to join an organization
func (s *server) InviteColleagues(ctx context.Context, in *invite.InviteColleaguesRequest) (*invite.InviteColleaguesResponse, error) {
	if in.OrganizationEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "OrganizationEntityID is required")
	}
	if in.InviterEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "InviterEntityID is required")
	}
	// Validate all colleague information
	for _, c := range in.Colleagues {
		if c.Email == "" {
			return nil, grpc.Errorf(codes.InvalidArgument, "Email is required")
		}
		if !validate.Email(c.Email) {
			return nil, grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("Email '%s' is invalid", c.Email))
		}
		if c.PhoneNumber == "" {
			return nil, grpc.Errorf(codes.InvalidArgument, "Phone number is required")
		}
		pn, err := phone.ParseNumber(c.PhoneNumber)
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("Phone number '%s' is invalid", c.PhoneNumber))
		}
		c.PhoneNumber = pn.String()
	}

	// Lookup org to get name
	org, err := s.getOrg(ctx, in.OrganizationEntityID)
	if err != nil {
		return nil, err
	}

	// Lookup inviter to get name
	inviter, err := s.getInternalEntity(ctx, in.InviterEntityID)
	if err != nil {
		return nil, err
	}

	for _, c := range in.Colleagues {
		inviteClientDataJSON, err := clientdata.ColleagueInviteClientJSON(org, inviter, c.FirstName, "", "")
		if err != nil {
			return nil, errors.Trace(err)
		}
		s.processInvite(
			ctx,
			simpleTokenGenerator,
			org, inviter,
			"", "", c.Email, c.PhoneNumber, string(inviteClientDataJSON),
			models.ColleagueInvite,
			invite.VERIFICATION_REQUIREMENT_PHONE_MATCH, []inviteDeliveryChannel{inviteDeliveryEmail},
			nil)
	}

	events.Publish(s.sns, s.eventsTopic, events.Service_INVITE, &invite.Event{
		Type: invite.Event_INVITED_COLLEAGUES,
		Details: &invite.Event_InvitedColleagues{
			InvitedColleagues: &invite.InvitedColleagues{
				OrganizationEntityID: in.OrganizationEntityID,
				InviterEntityID:      in.InviterEntityID,
			},
		},
	})

	return &invite.InviteColleaguesResponse{}, nil
}

// InvitePatients sends invites to people to join an organization
func (s *server) InvitePatients(ctx context.Context, in *invite.InvitePatientsRequest) (*invite.InvitePatientsResponse, error) {
	if in.OrganizationEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "OrganizationEntityID is required")
	}
	if in.InviterEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "InviterEntityID is required")
	}
	// Validate all patient information
	for _, p := range in.Patients {
		if p.PhoneNumber == "" {
			return nil, grpc.Errorf(codes.InvalidArgument, "Phone number is required")
		}
		pn, err := phone.ParseNumber(p.PhoneNumber)
		if err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, fmt.Sprintf("Phone number '%s' is invalid", p.PhoneNumber))
		}
		p.PhoneNumber = pn.String()
	}

	// Lookup org to get name
	org, err := s.getOrg(ctx, in.OrganizationEntityID)
	if err != nil {
		return nil, err
	}

	settingsRes, err := s.settingsClient.GetValues(ctx, &settings.GetValuesRequest{
		NodeID: in.OrganizationEntityID,
		Keys: []*settings.ConfigKey{
			{
				Key: invite.ConfigKeyTwoFactorVerificationForSecureConversation,
			},
			{
				Key: invite.ConfigKeyPatientInviteChannelPreference,
			},
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	requirePhoneAndEmailForSecureConversationCreation := settingsRes.Values[0].GetBoolean()
	inviteDeliveryPreference := settingsRes.Values[1].GetSingleSelect()

	// Lookup inviter to get name
	var inviter *directory.Entity
	if in.InviterEntityID != "" {
		inviter, err = s.getInternalEntity(ctx, in.InviterEntityID)
		if err != nil {
			return nil, err
		}
	}

	for _, p := range in.Patients {
		inviteClientDataJSON, err := clientdata.PatientInviteClientJSON(org, "", "", invite.LOOKUP_INVITE_RESPONSE_PATIENT)
		if err != nil {
			return nil, errors.Trace(err)
		}

		var deliveryChannels []inviteDeliveryChannel
		var verificationRequirement invite.InviteVerificationRequirement
		if requirePhoneAndEmailForSecureConversationCreation.Value {
			if !environment.IsProd() {
				deliveryChannels = append(deliveryChannels, inviteDeliveryEmail)
				verificationRequirement = invite.VERIFICATION_REQUIREMENT_PHONE_MATCH
			} else {
				deliveryChannels = append(deliveryChannels, inviteDeliverySMS)
				verificationRequirement = invite.VERIFICATION_REQUIREMENT_PHONE
			}
		} else {
			verificationRequirement = invite.VERIFICATION_REQUIREMENT_PHONE
			if p.Email != "" && p.PhoneNumber != "" {
				if inviteDeliveryPreference.Item.ID == invite.PatientInviteChannelPreferenceEmail {
					deliveryChannels = append(deliveryChannels, inviteDeliveryEmail)
				} else if inviteDeliveryPreference.Item.ID == invite.PatientInviteChannelPreferenceSMS {
					deliveryChannels = append(deliveryChannels, inviteDeliverySMS)
				}
			} else if p.Email != "" {
				deliveryChannels = append(deliveryChannels, invite.PatientInviteChannelPreferenceEmail)
			} else if p.PhoneNumber != "" {
				deliveryChannels = append(deliveryChannels, invite.PatientInviteChannelPreferenceSMS)
			}
		}

		s.processInvite(
			ctx,
			simpleTokenGenerator,
			org, inviter,
			p.FirstName, p.ParkedEntityID, p.Email, p.PhoneNumber, string(inviteClientDataJSON),
			models.PatientInvite,
			verificationRequirement,
			deliveryChannels,
			nil)
	}

	events.Publish(s.sns, s.eventsTopic, events.Service_INVITE, &invite.Event{
		Type: invite.Event_INVITED_PATIENTS,
		Details: &invite.Event_InvitedPatients{
			InvitedPatients: &invite.InvitedPatients{
				OrganizationEntityID: in.OrganizationEntityID,
				InviterEntityID:      in.InviterEntityID,
			},
		},
	})
	return &invite.InvitePatientsResponse{}, err
}

func (s *server) processInvite(
	ctx context.Context,
	tokenGenerator common.TokenGenerator,
	org, inviter *directory.Entity,
	firstName, parkedEntityID, email, phoneNumber, inviteClientDataStr string,
	inviteType models.InviteType,
	verificationRequirement invite.InviteVerificationRequirement,
	deliveryChannels []inviteDeliveryChannel,
	additionalValues map[string]string) error {
	// TODO: enqueue invite rather than sending directly
	var token, inviteURL string
	var err error
	for retry := 0; retry < 5; retry++ {
		token, err = tokenGenerator.GenerateToken()
		if err != nil {
			return errors.Trace(err)
		}
		values := map[string]string{
			"invite_token": token,
			"client_data":  inviteClientDataStr,
			"invite_type":  string(inviteType),
		}
		for k, v := range additionalValues {
			values[k] = v
		}
		if s.webInviteURL != nil {
			// Close the URL to avoid modifying the template
			ur := *s.webInviteURL
			query := ur.Query()
			query.Add("invite", token)
			ur.RawQuery = query.Encode()
			values[branch.DesktopURL] = ur.String()
		}
		attr := make(map[string]interface{}, len(values))
		for k, v := range values {
			attr[k] = v
		}
		inviteURL, err = s.branch.URL(attr)
		if err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to generate branch URL: %s", err)
			continue
		}
		pn := phoneNumber
		emailForInvite := email
		// We do not store phone numbers or emails for patients in dynamodb since it doesn't support simple encryption at rest
		if inviteType == models.PatientInvite {
			// We can't have these being empty attributes so populate them with informative info
			pn = phiAttributeText
			emailForInvite = phiAttributeText
		}

		var inviterID string
		if inviter != nil {
			inviterID = inviter.ID
		}

		vr, err := verificationRequirementFromRequest(verificationRequirement)
		if err != nil {
			return errors.Trace(err)
		}

		err = s.dal.InsertInvite(ctx, &models.Invite{
			Token:                   token,
			Type:                    inviteType,
			OrganizationEntityID:    org.ID,
			InviterEntityID:         inviterID,
			Email:                   emailForInvite,
			PhoneNumber:             pn,
			Created:                 s.clk.Now(),
			URL:                     inviteURL,
			ParkedEntityID:          parkedEntityID,
			Values:                  values,
			VerificationRequirement: vr,
		})
		if err == nil {
			break
		}
		if errors.Cause(err) != dal.ErrDuplicateInviteToken {
			golog.ContextLogger(ctx).Errorf("Failed to insert invite: %s", err)
			return nil
		}
	}

	switch inviteType {
	case models.ColleagueInvite:
		if err := s.sendColleagueOutbound(ctx, email, inviteURL, token, org, inviter); err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to send colleague invite outbound comms: %s", err)
		}
	case models.PatientInvite:
		if err := s.sendPatientOutbound(ctx, phoneNumber, email, inviteURL, token, org, deliveryChannels); err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to send patient invite outbound comms: %s", err)
		}
	default:
		golog.ContextLogger(ctx).Errorf("Unknown invite type %s. No outbound message sent.", inviteType)
	}
	return nil
}

func (s *server) sendColleagueOutbound(ctx context.Context, email, inviteURL, token string, org, inviter *directory.Entity) error {
	if _, err := s.excommsClient.SendMessage(ctx, &excomms.SendMessageRequest{
		DeprecatedChannel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          fmt.Sprintf("Invite to join %s on Spruce", org.Info.DisplayName),
				FromName:         "Spruce",
				FromEmailAddress: s.fromEmail,
				Body:             fmt.Sprintf("Your invite link is %s [%s]", inviteURL, token),
				ToEmailAddress:   email,
				Transactional:    true,
				TemplateID:       s.colleagueInviteTemplateID,
				TemplateSubstitutions: []*excomms.EmailMessage_Substitution{
					{Key: "{orgname}", Value: org.Info.DisplayName},
					{Key: "{inviteurl}", Value: inviteURL},
					{Key: "{invitername}", Value: inviter.Info.DisplayName},
					{Key: "{invitecode}", Value: token},
				},
			},
		},
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (s *server) sendPatientOutbound(
	ctx context.Context,
	phoneNumber, email, inviteURL, token string,
	org *directory.Entity,
	deliveryChannels []inviteDeliveryChannel) error {

	channelsDeliveredOn := make(map[inviteDeliveryChannel]struct{}, len(deliveryChannels))
	for _, deliveryChannel := range deliveryChannels {

		if _, ok := channelsDeliveredOn[deliveryChannel]; ok {
			continue
		}

		switch deliveryChannel {
		case inviteDeliveryEmail:
			if _, err := s.excommsClient.SendMessage(ctx, &excomms.SendMessageRequest{
				DeprecatedChannel: excomms.ChannelType_EMAIL,
				Message: &excomms.SendMessageRequest_Email{
					Email: &excomms.EmailMessage{
						Subject:          fmt.Sprintf("Please join %s on Spruce", org.Info.DisplayName),
						FromName:         "Spruce",
						FromEmailAddress: s.fromEmail,
						Body:             fmt.Sprintf("Your invite link is %s [%s]", inviteURL, token),
						ToEmailAddress:   email,
						Transactional:    true,
						TemplateID:       s.patientInviteTemplateID,
						TemplateSubstitutions: []*excomms.EmailMessage_Substitution{
							{Key: "{orgname}", Value: org.Info.DisplayName},
							{Key: "{inviteurl}", Value: inviteURL},
							{Key: "{invitecode}", Value: token},
						},
					},
				},
			}); err != nil {
				return errors.Trace(err)
			}

		case inviteDeliverySMS:
			msgText := fmt.Sprintf("%s has invited you to use Spruce Care Messenger. %s [If prompted, your provider code is %s]", org.Info.DisplayName, inviteURL, token)
			if _, err := s.excommsClient.SendMessage(ctx, &excomms.SendMessageRequest{
				DeprecatedChannel: excomms.ChannelType_SMS,
				Message: &excomms.SendMessageRequest_SMS{
					SMS: &excomms.SMSMessage{
						Text:            msgText,
						FromPhoneNumber: s.fromNumber,
						ToPhoneNumber:   phoneNumber,
					},
				},
			}); err != nil {
				return errors.Trace(err)
			}
		}

		channelsDeliveredOn[deliveryChannel] = struct{}{}
	}

	return nil
}

func (s *server) getEntity(ctx context.Context, entityID string) (*directory.Entity, error) {
	// Lookup organization to get name
	res, err := s.directoryClient.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	})
	if err != nil {
		switch grpc.Code(err) {
		case codes.NotFound:
			return nil, errors.Errorf("EntityID %s not found", entityID)
		}
		return nil, errors.Trace(err)
	}
	// Sanity check
	if len(res.Entities) != 1 {
		return nil, errors.Errorf("Expected 1 entity got %d", len(res.Entities))
	}
	return res.Entities[0], nil
}

func (s *server) getInternalEntity(ctx context.Context, entityID string) (*directory.Entity, error) {
	entity, err := s.getEntity(ctx, entityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if entity.Type != directory.EntityType_INTERNAL {
		return nil, grpc.Errorf(codes.InvalidArgument, "entityID %s is not an internal entity", entityID)
	}
	return entity, nil
}

func (s *server) getOrg(ctx context.Context, orgID string) (*directory.Entity, error) {
	entity, err := s.getEntity(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if entity.Type != directory.EntityType_ORGANIZATION {
		return nil, grpc.Errorf(codes.InvalidArgument, "OrganizationEntityID %s not an organization", orgID)
	}
	return entity, nil
}

// LookupInvite returns information about an invite by token
func (s *server) LookupInvite(ctx context.Context, in *invite.LookupInviteRequest) (*invite.LookupInviteResponse, error) {
	var err error
	var inv *models.Invite
	switch lookupKey := in.LookupKeyOneof.(type) {
	case *invite.LookupInviteRequest_Token:
		// Do our backwards compatible mapping till we can get rid of this switch
		token := lookupKey.Token
		if in.InviteToken != "" {
			token = in.InviteToken
		}
		inv, err = s.lookupInviteForToken(ctx, token)
		if err != nil {
			return nil, errors.Trace(err)
		}
	case *invite.LookupInviteRequest_OrganizationEntityID:
		// TODO: Until we can remove this code path just return the first one we find
		invs, err := s.lookupInvitesForOrganization(ctx, lookupKey.OrganizationEntityID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if len(invs) != 0 {
			inv = invs[0]
		} else {
			return nil, grpc.Errorf(codes.NotFound, "No invites found for org %s", lookupKey.OrganizationEntityID)
		}
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unsupported lookup key type %T", lookupKey)
	}
	if inv.Type != models.ColleagueInvite && inv.Type != models.PatientInvite && inv.Type != models.OrganizationCodeInvite {
		return nil, errors.Errorf("unsupported invite type %s", string(inv.Type))
	}
	values := make([]*invite.AttributionValue, 0, len(inv.Values))
	for k, v := range inv.Values {
		values = append(values, &invite.AttributionValue{
			Key:   k,
			Value: v,
		})
	}

	verificationRequirement, err := verificationRequirementAsResponse(inv.VerificationRequirement)
	if err != nil {
		return nil, errors.Errorf("unknown verification requirement for %s: %s", inv.Token, err)
	}
	resp := &invite.LookupInviteResponse{Values: values}
	switch inv.Type {
	case models.ColleagueInvite:
		resp.Type = invite.LOOKUP_INVITE_RESPONSE_COLLEAGUE
		resp.Invite = &invite.LookupInviteResponse_Colleague{
			Colleague: &invite.ColleagueInvite{
				OrganizationEntityID: inv.OrganizationEntityID,
				InviterEntityID:      inv.InviterEntityID,
				Colleague: &invite.Colleague{
					Email:       inv.Email,
					PhoneNumber: inv.PhoneNumber,
				},
			},
		}
	case models.PatientInvite:
		resp.Type = invite.LOOKUP_INVITE_RESPONSE_PATIENT
		resp.Invite = &invite.LookupInviteResponse_Patient{
			Patient: &invite.PatientInvite{
				OrganizationEntityID: inv.OrganizationEntityID,
				InviterEntityID:      inv.InviterEntityID,
				Patient: &invite.Patient{
					ParkedEntityID: inv.ParkedEntityID,
					PhoneNumber:    inv.PhoneNumber,
				},
				InviteVerificationRequirement: verificationRequirement,
			},
		}
	case models.OrganizationCodeInvite:
		resp.Type = invite.LOOKUP_INVITE_RESPONSE_ORGANIZATION_CODE
		orgResp, err := organizationInviteAsResponse(inv)
		if err != nil {
			return nil, errors.Trace(err)
		}
		resp.Invite = &invite.LookupInviteResponse_Organization{
			Organization: orgResp,
		}
	}
	return resp, nil
}

func (s *server) LookupInvites(ctx context.Context, in *invite.LookupInvitesRequest) (*invite.LookupInvitesResponse, error) {
	var res invite.LookupInvitesResponse
	switch keyType := in.Key.(type) {
	case *invite.LookupInvitesRequest_ParkedEntityID:
		invites, err := s.dal.InvitesForParkedEntityID(ctx, keyType.ParkedEntityID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		patientInvitesList := invite.PatientInviteList{
			PatientInvites: make([]*invite.PatientInvite, len(invites)),
		}
		for i, inv := range invites {

			verificationRequirement, err := verificationRequirementAsResponse(inv.VerificationRequirement)
			if err != nil {
				return nil, errors.Errorf("unknown verfication requirement for invite %s: %s", inv.Token, err)
			}

			patientInvitesList.PatientInvites[i] = &invite.PatientInvite{
				OrganizationEntityID: inv.OrganizationEntityID,
				InviterEntityID:      inv.InviterEntityID,
				Patient: &invite.Patient{
					ParkedEntityID: inv.ParkedEntityID,
					PhoneNumber:    inv.PhoneNumber,
				},
				InviteVerificationRequirement: verificationRequirement,
			}
		}

		res.Type = invite.LOOKUP_INVITES_RESPONSE_PATIENT_LIST
		res.List = &invite.LookupInvitesResponse_PatientInviteList{
			PatientInviteList: &patientInvitesList,
		}
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unsupported lookup key type %s", in.LookupKeyType.String())
	}

	return &res, nil
}

// LookupOrganizationInvites returns the set of organization invites associated with an org id
func (s *server) LookupOrganizationInvites(ctx context.Context, in *invite.LookupOrganizationInvitesRequest) (*invite.LookupOrganizationInvitesResponse, error) {
	invs, err := s.lookupInvitesForOrganization(ctx, in.OrganizationEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	orgInvites := make([]*invite.OrganizationInvite, len(invs))
	for i, inv := range invs {
		oi, err := organizationInviteAsResponse(inv)
		if err != nil {
			return nil, errors.Wrap(err, "failed to transform org invite to response")
		}
		orgInvites[i] = oi
	}
	return &invite.LookupOrganizationInvitesResponse{OrganizationInvites: orgInvites}, nil
}

func (s *server) lookupInvitesForOrganization(ctx context.Context, orgEntityID string) ([]*models.Invite, error) {
	tokens, err := s.dal.TokensForEntity(ctx, orgEntityID)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpc.Errorf(codes.NotFound, "Invite not found for entity "+orgEntityID)
		}
		return nil, errors.Trace(err)
	}
	invites := make([]*models.Invite, len(tokens))
	for i, t := range tokens {
		inv, err := s.lookupInviteForToken(ctx, t)
		if err != nil {
			return nil, errors.Trace(err)
		}
		invites[i] = inv
	}
	return invites, nil
}

// ModifyOrganizationInvite modifies the specified organization invite
func (s *server) ModifyOrganizationInvite(ctx context.Context, in *invite.ModifyOrganizationInviteRequest) (*invite.ModifyOrganizationInviteResponse, error) {
	if in.Token == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Token required")
	}
	inv, err := s.dal.UpdateInvite(ctx, in.Token, &models.InviteUpdate{
		Tags: in.Tags,
	})
	if errors.Cause(err) == dal.ErrNotFound {
		return nil, grpc.Errorf(codes.NotFound, "Invite not found with token %s", in.Token)
	} else if err != nil {
		return nil, errors.Wrapf(err, "failed to update invite with code %s", in.Token)
	}
	oi, err := organizationInviteAsResponse(inv)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transform org invite to response")
	}
	return &invite.ModifyOrganizationInviteResponse{OrganizationInvite: oi}, nil
}

func (s *server) lookupInviteForToken(ctx context.Context, token string) (*models.Invite, error) {
	inv, err := s.dal.InviteForToken(ctx, token)
	if err != nil {
		if errors.Cause(err) == dal.ErrNotFound {
			return nil, grpc.Errorf(codes.NotFound, "Invite not found with token "+token)
		}
		return nil, errors.Trace(err)
	}
	return inv, nil
}

// MarkInviteConsumed deletes the associated invite and records it's consumption
func (s *server) MarkInviteConsumed(ctx context.Context, in *invite.MarkInviteConsumedRequest) (*invite.MarkInviteConsumedResponse, error) {
	// TODO: Record consumption metrics
	if err := s.dal.DeleteInvite(ctx, in.Token); err != nil {
		return nil, errors.Trace(err)
	}
	return &invite.MarkInviteConsumedResponse{}, nil
}

// DeleteInvite deletes an invite based on the key.
func (s *server) DeleteInvite(ctx context.Context, in *invite.DeleteInviteRequest) (*invite.DeleteInviteResponse, error) {
	var tokens []string
	switch keyType := in.Key.(type) {
	case *invite.DeleteInviteRequest_Token:
		tokens = []string{keyType.Token}
	case *invite.DeleteInviteRequest_ParkedEntityID:

		invites, err := s.dal.InvitesForParkedEntityID(ctx, keyType.ParkedEntityID)
		if err != nil {
			return nil, errors.Errorf("unable to get invites for parkedEntityID %s : %s", keyType.ParkedEntityID, err)
		}

		for _, inv := range invites {
			tokens = append(tokens, inv.Token)
		}

	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "unknown delete key %s", in.DeleteInviteKey)
	}

	for _, token := range tokens {
		if err := s.dal.DeleteInvite(ctx, token); err != nil {
			return nil, errors.Errorf("unable to delete invite for token %s : %s", token, err.Error())
		}
	}

	return &invite.DeleteInviteResponse{}, nil
}

// SetAttributionData associate attribution data with a device
func (s *server) SetAttributionData(ctx context.Context, in *invite.SetAttributionDataRequest) (*invite.SetAttributionDataResponse, error) {
	if in.DeviceID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "DeviceID is required")
	}
	values := make(map[string]string, len(in.Values))
	for _, v := range in.Values {
		values[v.Key] = v.Value
	}
	if err := s.dal.SetAttributionData(ctx, in.DeviceID, values); err != nil {
		return nil, errors.Trace(err)
	}
	return &invite.SetAttributionDataResponse{}, nil
}

// CreateOrganizationInvite creates an invite code for the organization
func (s *server) CreateOrganizationInvite(ctx context.Context, in *invite.CreateOrganizationInviteRequest) (*invite.CreateOrganizationInviteResponse, error) {
	if in.OrganizationEntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Organization Entity ID is required")
	}

	// Lookup org to get name
	org, err := s.getOrg(ctx, in.OrganizationEntityID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	inviteClientDataJSON, err := clientdata.PatientInviteClientJSON(org, "", "", invite.LOOKUP_INVITE_RESPONSE_ORGANIZATION_CODE)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var token string
	for retry := 0; retry < 5; retry++ {
		token, err = simpleTokenGenerator.GenerateToken()
		if err != nil {
			return nil, errors.Errorf("Error while generating org code: %s", err)
		}
		values := map[string]string{
			"invite_token": token,
			"client_data":  inviteClientDataJSON,
			"invite_type":  string(models.OrganizationCodeInvite),
		}
		if s.webInviteURL != nil {
			// Close the URL to avoid modifying the template
			ur := *s.webInviteURL
			query := ur.Query()
			query.Add("invite", token)
			ur.RawQuery = query.Encode()
			values[branch.DesktopURL] = ur.String()
		}
		attr := make(map[string]interface{}, len(values))
		for k, v := range values {
			attr[k] = v
		}
		inviteURL, err := s.branch.URL(attr)
		if err != nil {
			golog.ContextLogger(ctx).Errorf("Failed to generate branch URL: %s", err)
			token = ""
			continue
		}
		if err := s.dal.InsertEntityToken(ctx, in.OrganizationEntityID, token); errors.Cause(err) == dal.ErrDuplicateInviteToken {
			token = ""
			continue
		} else if err != nil {
			return nil, errors.Errorf("Failed to insert entity token: %s", err)
		}
		err = s.dal.InsertInvite(ctx, &models.Invite{
			Token:                token,
			Type:                 models.OrganizationCodeInvite,
			OrganizationEntityID: in.OrganizationEntityID,
			Created:              s.clk.Now(),
			URL:                  inviteURL,
			Values:               values,
		})
		if err == nil {
			break
		} else if errors.Cause(err) == dal.ErrDuplicateInviteToken {
			token = ""
			continue
		} else if err != nil {
			return nil, errors.Errorf("Failed to insert organization invite: %s", err)
		}
	}
	if token == "" {
		return nil, errors.Errorf("Failed to generate branch link and code")
	}
	return &invite.CreateOrganizationInviteResponse{
		Organization: &invite.OrganizationInvite{
			OrganizationEntityID: in.OrganizationEntityID,
			Token:                token,
		},
	}, nil
}
