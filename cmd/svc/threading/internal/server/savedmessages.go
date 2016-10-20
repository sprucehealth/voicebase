package server

import (
	"context"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc/codes"
)

func (s *threadsServer) CreateSavedMessage(ctx context.Context, in *threading.CreateSavedMessageRequest) (*threading.CreateSavedMessageResponse, error) {
	if in.OwnerEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "OwnerEntityID is required")
	}
	if in.CreatorEntityID == "" {
		return nil, grpcErrorf(codes.InvalidArgument, "CreatorEntityID is required")
	}
	msg := in.GetMessage()
	textRefs, err := processMessagePost(msg)
	if err != nil {
		return nil, err
	}

	attachments, err := transformAttachmentsFromRequest(msg.Attachments)
	if err != nil {
		return nil, errors.Trace(err)
	}
	mediaIDs := mediaIDsFromAttachments(attachments)

	sm := &models.SavedMessage{
		Title:           in.Title,
		OrganizationID:  in.OrganizationID,
		CreatorEntityID: in.CreatorEntityID,
		OwnerEntityID:   in.OwnerEntityID,
		Internal:        msg.Internal,
		Content: &models.Message{
			Text:        msg.Text,
			Title:       msg.Title,
			Summary:     msg.Summary,
			Status:      models.MESSAGE_STATUS_NORMAL,
			TextRefs:    textRefs,
			Attachments: attachments,
		},
	}

	id, err := s.dal.CreateSavedMessage(ctx, sm)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(mediaIDs) > 0 {
		_, err = s.mediaClient.ClaimMedia(ctx, &media.ClaimMediaRequest{
			MediaIDs:  mediaIDs,
			OwnerType: media.MediaOwnerType_SAVED_MESSAGE,
			OwnerID:   id.String(),
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	// Refetch the saved messages to get the actual version from the database (timestamps can change due to precision)
	sms, err := s.dal.SavedMessages(ctx, []models.SavedMessageID{id})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(sms) == 0 {
		return nil, errors.Errorf("newly created saved message %s not found", id)
	}
	sm = sms[0]
	smr, err := transformSavedMessageToResponse(sm)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.CreateSavedMessageResponse{SavedMessage: smr}, nil
}

func (s *threadsServer) DeleteSavedMessage(ctx context.Context, in *threading.DeleteSavedMessageRequest) (*threading.DeleteSavedMessageResponse, error) {
	id, err := models.ParseSavedMessageID(in.SavedMessageID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid saved message ID %s", id)
	}
	if _, err := s.dal.DeleteSavedMessages(ctx, []models.SavedMessageID{id}); err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.DeleteSavedMessageResponse{}, nil
}

func (s *threadsServer) SavedMessages(ctx context.Context, in *threading.SavedMessagesRequest) (*threading.SavedMessagesResponse, error) {
	var sms []*models.SavedMessage
	switch by := in.By.(type) {
	case *threading.SavedMessagesRequest_IDs:
		if len(by.IDs.IDs) == 0 {
			return nil, grpcErrorf(codes.InvalidArgument, "Empty ID list")
		}
		ids := make([]models.SavedMessageID, len(by.IDs.IDs))
		for i, strID := range by.IDs.IDs {
			id, err := models.ParseSavedMessageID(strID)
			if err != nil {
				return nil, grpcErrorf(codes.InvalidArgument, "Invalid saved message ID %s", strID)
			}
			ids[i] = id
		}
		var err error
		sms, err = s.dal.SavedMessages(ctx, ids)
		if err != nil {
			return nil, errors.Trace(err)
		}
	case *threading.SavedMessagesRequest_EntityIDs:
		if len(by.EntityIDs.IDs) == 0 {
			return nil, grpcErrorf(codes.InvalidArgument, "Empty entity ID list")
		}
		var err error
		sms, err = s.dal.SavedMessagesForEntities(ctx, by.EntityIDs.IDs)
		if err != nil {
			return nil, errors.Trace(err)
		}
	default:
		return nil, grpcErrorf(codes.InvalidArgument, "Missing By")
	}
	res := &threading.SavedMessagesResponse{
		SavedMessages: make([]*threading.SavedMessage, len(sms)),
	}
	for i, sm := range sms {
		var err error
		res.SavedMessages[i], err = transformSavedMessageToResponse(sm)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	return res, nil
}

func (s *threadsServer) UpdateSavedMessage(ctx context.Context, in *threading.UpdateSavedMessageRequest) (*threading.UpdateSavedMessageResponse, error) {
	id, err := models.ParseSavedMessageID(in.SavedMessageID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Invalid saved message ID %s", id)
	}
	update := &dal.SavedMessageUpdate{}
	if in.Title != "" {
		update.Title = &in.Title
	}
	if msg := in.GetMessage(); msg != nil {
		textRefs, err := processMessagePost(msg)
		if err != nil {
			return nil, err
		}
		attachments, err := transformAttachmentsFromRequest(msg.Attachments)
		if err != nil {
			return nil, errors.Trace(err)
		}
		mediaIDs := mediaIDsFromAttachments(attachments)

		update.Content = &models.Message{
			Text:        msg.Text,
			Title:       msg.Title,
			Summary:     msg.Summary,
			Status:      models.MESSAGE_STATUS_NORMAL,
			TextRefs:    textRefs,
			Attachments: attachments,
		}
		if len(mediaIDs) > 0 {
			_, err = s.mediaClient.ClaimMedia(ctx, &media.ClaimMediaRequest{
				MediaIDs:  mediaIDs,
				OwnerType: media.MediaOwnerType_SAVED_MESSAGE,
				OwnerID:   id.String(),
			})
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}
	if err := s.dal.UpdateSavedMessage(ctx, id, update); err != nil {
		return nil, errors.Trace(err)
	}

	// Refetch the saved messages to get the actual version from the database (timestamps can change due to precision)
	sms, err := s.dal.SavedMessages(ctx, []models.SavedMessageID{id})
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(sms) == 0 {
		return nil, errors.Errorf("updated saved message %s not found", id)
	}
	smr, err := transformSavedMessageToResponse(sms[0])
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &threading.UpdateSavedMessageResponse{SavedMessage: smr}, nil
}

func transformSavedMessageToResponse(sm *models.SavedMessage) (*threading.SavedMessage, error) {
	tsm := &threading.SavedMessage{
		ID:              sm.ID.String(),
		Title:           sm.Title,
		OrganizationID:  sm.OrganizationID,
		CreatorEntityID: sm.CreatorEntityID,
		OwnerEntityID:   sm.OwnerEntityID,
		Internal:        sm.Internal,
		Created:         uint64(sm.Created.Unix()),
		Modified:        uint64(sm.Modified.Unix()),
	}
	switch v := sm.Content.(type) {
	case *models.Message:
		m, err := TransformMessageToResponse(v)
		if err != nil {
			return nil, errors.Trace(err)
		}
		tsm.Content = &threading.SavedMessage_Message{Message: m}
	default:
		return nil, errors.Errorf("unknown saved message content type %T", sm.Content)
	}
	return tsm, nil
}
