package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"context"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/twilio"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc/codes"
)

const (
	ipCallTokenTTL = 6 * 60 * 60
	ipCallTimeout  = 2 * time.Minute
)

type ipCallStateTransition struct {
	from, to models.IPCallState
}

var validIPCallParicipantStateTransitions = map[ipCallStateTransition]struct{}{
	{from: models.IPCallStatePending, to: models.IPCallStateAccepted}:    {},
	{from: models.IPCallStatePending, to: models.IPCallStateDeclined}:    {},
	{from: models.IPCallStatePending, to: models.IPCallStateFailed}:      {}, // gives client the ability to automatically fail a call in the event of wifi being disabled on the recipient side.
	{from: models.IPCallStateAccepted, to: models.IPCallStateConnected}:  {},
	{from: models.IPCallStateAccepted, to: models.IPCallStateFailed}:     {},
	{from: models.IPCallStateAccepted, to: models.IPCallStateCompleted}:  {}, // hanging up after accepting but before connecting
	{from: models.IPCallStateConnected, to: models.IPCallStateFailed}:    {},
	{from: models.IPCallStateConnected, to: models.IPCallStateCompleted}: {},
}

func (e *excommsService) InitiateIPCall(ctx context.Context, req *excomms.InitiateIPCallRequest) (*excomms.InitiateIPCallResponse, error) {
	// For now only allow 2 party calls
	if len(req.RecipientEntityIDs) != 1 {
		return nil, grpcErrorf(codes.InvalidArgument, "Must provide exactly one participant")
	}
	if req.RecipientEntityIDs[0] == req.CallerEntityID {
		return nil, grpcErrorf(codes.InvalidArgument, "Recipient may not be the same entity as the caller")
	}

	entityIDs := append(req.RecipientEntityIDs, req.CallerEntityID)
	leres, err := e.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
			BatchEntityID: &directory.IDList{IDs: entityIDs},
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	})
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	if len(leres.Entities) != len(entityIDs) {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to find all entities")
	}

	call := &models.IPCall{Pending: true}
	switch req.Type {
	case excomms.IPCallType_VIDEO:
		call.Type = models.IPCallTypeVideo
	case excomms.IPCallType_AUDIO:
		call.Type = models.IPCallTypeAudio
	default:
		return nil, grpcErrorf(codes.InvalidArgument, "Unknown call type %s", req.Type.String())
	}

	call.Participants = make([]*models.IPCallParticipant, 0, len(leres.Entities))
	var callerPar *models.IPCallParticipant
	var org *directory.Entity
	for _, ent := range leres.Entities {
		var o *directory.Entity
		for _, m := range ent.Memberships {
			if m.Type == directory.EntityType_ORGANIZATION {
				o = m
				break
			}
		}
		if o == nil {
			return nil, grpcErrorf(codes.InvalidArgument, "Participant %s does not belong to any organizations", ent.ID)
		}
		if org == nil {
			org = o
		} else if org.ID != o.ID {
			// As a sanity check make sure everyone involved belongs to the same org.
			return nil, grpcErrorf(codes.InvalidArgument, "All participants must belong to the same organization")
		}
		if ent.AccountID == "" {
			return nil, grpcErrorf(codes.InvalidArgument, "Participant %s missing account ID", ent.ID)
		}
		p := &models.IPCallParticipant{
			EntityID:  ent.ID,
			AccountID: ent.AccountID,
		}
		p.Identity, err = e.genIPCallIdentity()
		if err != nil {
			return nil, grpcErrorf(codes.Internal, "Failed to generate identity: %s", err)
		}
		if ent.ID == req.CallerEntityID {
			p.Role = models.IPCallParticipantRoleCaller
			p.State = models.IPCallStateAccepted
			callerPar = p
			p.NetworkType, err = transformNetworkTypeToModel(req.NetworkType)
			if err != nil {
				return nil, grpcErrorf(codes.InvalidArgument, err.Error())
			}
		} else {
			p.Role = models.IPCallParticipantRoleRecipient
			p.State = models.IPCallStatePending
			p.NetworkType = models.NetworkTypeUnknown
		}
		call.Participants = append(call.Participants, p)
	}

	if err := e.dal.CreateIPCall(ctx, call); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	notificationMsgs := make(map[string]string, len(req.RecipientEntityIDs))
	for _, eid := range req.RecipientEntityIDs {
		notificationMsgs[eid] = "☎️ Video call from your healthcare provider"
	}
	if err := e.notificationClient.SendNotification(&notification.Notification{
		Type:             notification.IncomingIPCall,
		CallID:           call.ID.String(),
		OrganizationID:   org.ID,
		EntitiesToNotify: req.RecipientEntityIDs,
		DedupeKey:        call.ID.String(),
		CollapseKey:      string(notification.IncomingIPCall),
		ShortMessages:    notificationMsgs,
	}); err != nil {
		golog.Errorf("Failed to send notification about new IP call: %s", err)
	}

	rcall, err := e.transformIPCallToResponse(call, callerPar)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &excomms.InitiateIPCallResponse{Call: rcall}, nil
}

func (e *excommsService) IPCall(ctx context.Context, req *excomms.IPCallRequest) (*excomms.IPCallResponse, error) {
	if req.IPCallID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "IPCallID required")
	}
	id, err := models.ParseIPCallID(req.IPCallID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid IPCallID")
	}
	call, err := e.dal.IPCall(ctx, id)
	if errors.Cause(err) == dal.ErrIPCallNotFound {
		return nil, grpcErrorf(codes.NotFound, "IPCall %s not found", id)
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	if call.Pending && e.clock.Now().Sub(call.InitiatedTime) > ipCallTimeout {
		if err := e.timeoutIPCall(ctx, call); err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
	}
	// Find the participating account to be able to generate a proper token
	var par *models.IPCallParticipant
	for _, p := range call.Participants {
		if p.AccountID == req.AccountID {
			par = p
			break
		}
	}
	if par == nil {
		return nil, grpcErrorf(codes.PermissionDenied, "Account %s is not a participant in call %s", req.AccountID, call.ID)
	}
	rcall, err := e.transformIPCallToResponse(call, par)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &excomms.IPCallResponse{Call: rcall}, nil
}

func (e *excommsService) PendingIPCalls(ctx context.Context, req *excomms.PendingIPCallsRequest) (*excomms.PendingIPCallsResponse, error) {
	if req.AccountID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "AccountID required")
	}
	calls, err := e.dal.PendingIPCallsForAccount(ctx, req.AccountID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	res := &excomms.PendingIPCallsResponse{
		Calls: make([]*excomms.IPCall, 0, len(calls)),
	}
	for _, c := range calls {
		// Lazily timeout pending calls
		if e.clock.Now().Sub(c.InitiatedTime) > ipCallTimeout {
			if err := e.timeoutIPCall(ctx, c); err != nil {
				return nil, grpcErrorf(codes.Internal, err.Error())
			}
			continue
		}

		var par *models.IPCallParticipant
		for _, p := range c.Participants {
			if p.AccountID == req.AccountID {
				par = p
				break
			}
		}
		if par == nil {
			// Sanity check, this is an internal consistency error since the pending calls should only include calls with the account as a participant
			return nil, grpcErrorf(codes.Internal, "Participant not found for account %s even though call %s was returned", req.AccountID, c.ID)
		}
		call, err := e.transformIPCallToResponse(c, par)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
		res.Calls = append(res.Calls, call)
	}
	return res, nil
}

func (e *excommsService) UpdateIPCall(ctx context.Context, req *excomms.UpdateIPCallRequest) (*excomms.UpdateIPCallResponse, error) {
	if req.IPCallID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "IPCallID is required")
	}
	if req.AccountID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "AccountID is required")
	}
	callID, err := models.ParseIPCallID(req.IPCallID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "IPCallID is invalid")
	}
	newState, err := transformIPCallStateToModel(req.State)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	if newState == models.IPCallStatePending {
		return nil, grpcErrorf(codes.InvalidArgument, "Cannot transition to the PENDING State")
	}
	networkType, err := transformNetworkTypeToModel(req.NetworkType)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, err.Error())
	}
	var call *models.IPCall
	var par *models.IPCallParticipant
	var oldState models.IPCallState
	endOfCall := false
	err = e.dal.Transact(func(dl dal.DAL) error {
		call, err = dl.IPCall(ctx, callID, dal.ForUpdate)
		if errors.Cause(err) == dal.ErrIPCallNotFound {
			return grpcErrorf(codes.NotFound, "IPCall %s not found", callID)
		} else if err != nil {
			return grpcErrorf(codes.Internal, err.Error())
		}
		for _, p := range call.Participants {
			if p.AccountID == req.AccountID {
				par = p
				break
			}
		}
		if par == nil {
			return grpcErrorf(codes.PermissionDenied, "Account %s not a participant in %s", req.AccountID, callID)
		}
		if newState == par.State {
			// Nothing to do
			return nil
		}
		callWasActive := call.Active()
		// Validate that the new state is a valid transition from the current state
		if _, ok := validIPCallParicipantStateTransitions[ipCallStateTransition{from: par.State, to: newState}]; !ok {
			return grpcErrorf(codes.InvalidArgument, "Cannot transition from state %s to %s for %s", par.State, newState, par.EntityID)
		}
		// Update the participant so we don't have to refetch when returning the response
		oldState = par.State
		par.State = newState
		par.NetworkType = networkType
		if err := dl.UpdateIPCallParticipant(ctx, callID, req.AccountID, &dal.IPCallParticipantUpdate{State: &newState, NetworkType: &networkType}); err != nil {
			return errors.Trace(err)
		}
		if call.Pending && !newState.Pending() {
			update := &dal.IPCallUpdate{
				Pending: ptr.Bool(false),
			}
			call.Pending = false
			if newState == models.IPCallStateConnected {
				call.ConnectedTime = ptr.Time(e.clock.Now())
				update.ConnectedTime = call.ConnectedTime
			}
			if err := dl.UpdateIPCall(ctx, callID, update); err != nil {
				return errors.Trace(err)
			}
		}
		// If the call is still active and the new state is a terminal (end of call) state then trigger any actions (e.g. message)
		if callWasActive && newState.Terminal() {
			endOfCall = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if endOfCall {
		// Post message into thread if the receiving entity is a primary on a thread
		if err := e.postIPCallMessage(ctx, call); err != nil {
			// Too late to revert here so just log and move on
			golog.Errorf("Error creating post for IPCall: %s", err)
		}
	}
	rcall, err := e.transformIPCallToResponse(call, par)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &excomms.UpdateIPCallResponse{Call: rcall}, nil
}

func (e *excommsService) timeoutIPCall(ctx context.Context, call *models.IPCall) error {
	newParState := models.IPCallStateDeclined
	parUpdate := &dal.IPCallParticipantUpdate{State: &newParState}
	err := e.dal.Transact(func(dl dal.DAL) error {
		// Need to requery to lock the row
		c, err := dl.IPCall(ctx, call.ID, dal.ForUpdate)
		if err != nil {
			return errors.Trace(err)
		}
		// Update all participants to state declined
		for _, p := range call.Participants {
			p.State = newParState
			if err := dl.UpdateIPCallParticipant(ctx, call.ID, p.AccountID, parUpdate); err != nil {
				return errors.Trace(err)
			}
		}
		if c.Pending {
			call.Pending = false // Update the original call to match new value
			if err := dl.UpdateIPCall(ctx, c.ID, &dal.IPCallUpdate{Pending: ptr.Bool(false)}); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	})
	if err != nil {
		return errors.Trace(err)
	}
	// Post message into thread if the receiving entity is a primary on a thread
	if err := e.postIPCallMessage(ctx, call); err != nil {
		// Too late to revert here so just log and move on
		golog.Errorf("Error creating post for IPCall: %s", err)
	}
	return nil
}

func (e *excommsService) postIPCallMessage(ctx context.Context, call *models.IPCall) error {
	// Only know how to handle calls with 2 participants (caller and callee)
	if len(call.Participants) != 2 {
		return nil
	}

	var caller *models.IPCallParticipant
	var recipient *models.IPCallParticipant
	for _, p := range call.Participants {
		switch p.Role {
		case models.IPCallParticipantRoleRecipient:
			recipient = p
		case models.IPCallParticipantRoleCaller:
			caller = p
		}
	}

	res, err := e.threading.ThreadsForMember(ctx, &threading.ThreadsForMemberRequest{
		EntityID:    recipient.EntityID,
		PrimaryOnly: true,
	})
	if err != nil {
		return errors.Trace(err)
	}
	switch len(res.Threads) {
	default:
		return errors.Errorf("Expected 0 or 1 threads for primary entity %s, found %d", recipient.EntityID, len(res.Threads))
	case 0:
		return nil
	case 1:
	}
	thread := res.Threads[0]

	var title bml.BML
	var track *segment.Track
	if call.ConnectedTime != nil {
		dt := e.clock.Now().Sub(*call.ConnectedTime).Nanoseconds() / 1e9
		title = append(title, fmt.Sprintf("Video call, %d:%02ds", dt/60, dt%60))
		track = &segment.Track{
			Event: "video-visit-completed",
			Properties: map[string]interface{}{
				"duration":            fmt.Sprintf("%d:%02ds", dt/60, dt%60),
				"recipient_entity_id": recipient.EntityID,
			},
		}
	} else {
		title = append(title, "Video call, no answer")
		track = &segment.Track{
			Event: "video-visit-no-answer",
			Properties: map[string]interface{}{
				"recipient_entity_id": recipient.EntityID,
			},
		}
	}

	conc.Go(func() {
		entity, err := directory.SingleEntity(ctx, e.directory, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: caller.EntityID,
			},
		})
		if err != nil {
			golog.Errorf("Unable to get entity for %s : %s", entity.ID, err)
			return
		}

		track.UserId = entity.AccountID
		analytics.SegmentTrack(track)
	})

	titleText, err := title.Format()
	if err != nil {
		return errors.Trace(err)
	}
	summary, err := title.PlainText()
	if err != nil {
		return errors.Trace(err)
	}
	_, err = e.threading.PostMessage(ctx, &threading.PostMessageRequest{
		UUID:         call.ID.String(),
		ThreadID:     thread.ID,
		FromEntityID: caller.EntityID,
		Title:        titleText,
		Summary:      summary,
		DontNotify:   true,
	})
	return errors.Trace(err)
}

func (e *excommsService) transformIPCallToResponse(call *models.IPCall, par *models.IPCallParticipant) (*excomms.IPCall, error) {
	var token string
	var err error
	if par != nil && call.Pending {
		token, err = generateIPCallToken(par.Identity, e.twilioVideoConfigSID).ToJWT(e.twilioAccountSID, e.twilioSigningKeySID, e.twilioSigningKey)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	c := &excomms.IPCall{
		ID:           call.ID.String(),
		Pending:      call.Pending,
		Token:        token,
		Participants: make([]*excomms.IPCallParticipant, 0, len(call.Participants)),
	}
	switch call.Type {
	case models.IPCallTypeVideo:
		c.Type = excomms.IPCallType_VIDEO
	case models.IPCallTypeAudio:
		c.Type = excomms.IPCallType_AUDIO
	default:
		return nil, errors.Errorf("unknown call type %s for call %s", call.Type, call.ID)
	}
	for _, p := range call.Participants {
		cp := &excomms.IPCallParticipant{
			AccountID: p.AccountID,
			EntityID:  p.EntityID,
			Identity:  p.Identity,
		}
		switch p.Role {
		case models.IPCallParticipantRoleCaller:
			cp.Role = excomms.IPCallParticipantRole_CALLER
		case models.IPCallParticipantRoleRecipient:
			cp.Role = excomms.IPCallParticipantRole_RECIPIENT
		default:
			return nil, errors.Errorf("unknown role %s for ipcall %s participant account %s", p.Role, call.ID, p.AccountID)
		}
		switch p.State {
		case models.IPCallStateAccepted:
			cp.State = excomms.IPCallState_ACCEPTED
		case models.IPCallStateDeclined:
			cp.State = excomms.IPCallState_DECLINED
		case models.IPCallStateCompleted:
			cp.State = excomms.IPCallState_COMPLETED
		case models.IPCallStateConnected:
			cp.State = excomms.IPCallState_CONNECTED
		case models.IPCallStateFailed:
			cp.State = excomms.IPCallState_FAILED
		case models.IPCallStatePending:
			cp.State = excomms.IPCallState_PENDING
		default:
			return nil, errors.Errorf("unknown state %s for ipcall %s participant account %s", p.State, call.ID, p.AccountID)
		}
		switch p.NetworkType {
		case models.NetworkTypeUnknown:
			cp.NetworkType = excomms.NetworkType_UNKNOWN
		case models.NetworkTypeCellular:
			cp.NetworkType = excomms.NetworkType_CELLULAR
		case models.NetworkTypeWiFi:
			cp.NetworkType = excomms.NetworkType_WIFI
		default:
			return nil, errors.Errorf("unknown network type %s for ipcall %s participant account %s", p.NetworkType, call.ID, p.AccountID)
		}
		c.Participants = append(c.Participants, cp)
	}
	return c, nil
}

func transformIPCallStateToModel(state excomms.IPCallState) (models.IPCallState, error) {
	switch state {
	case excomms.IPCallState_ACCEPTED:
		return models.IPCallStateAccepted, nil
	case excomms.IPCallState_DECLINED:
		return models.IPCallStateDeclined, nil
	case excomms.IPCallState_COMPLETED:
		return models.IPCallStateCompleted, nil
	case excomms.IPCallState_CONNECTED:
		return models.IPCallStateConnected, nil
	case excomms.IPCallState_FAILED:
		return models.IPCallStateFailed, nil
	case excomms.IPCallState_PENDING:
		return models.IPCallStatePending, nil
	}
	return "", errors.Errorf("unknown ipcall state %s", state)
}

func transformNetworkTypeToModel(nt excomms.NetworkType) (models.NetworkType, error) {
	switch nt {
	case excomms.NetworkType_UNKNOWN:
		return models.NetworkTypeUnknown, nil
	case excomms.NetworkType_CELLULAR:
		return models.NetworkTypeCellular, nil
	case excomms.NetworkType_WIFI:
		return models.NetworkTypeWiFi, nil
	}
	return "", errors.Errorf("unknown network type %s", nt)
}

func generateIPCallToken(identity, configProfileSID string) *twilio.AccessToken {
	return &twilio.AccessToken{
		Identity: identity,
		Grants: []twilio.Grant{twilio.ConversationsGrant{
			ConfigurationProfileSID: configProfileSID,
		}},
		TTL: ipCallTokenTTL,
	}
}

func generateIPCallIdentity() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", errors.Trace(err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
