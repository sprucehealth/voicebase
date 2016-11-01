package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/threading"
)

func transformQueryFromRequest(q *threading.Query) (*models.Query, error) {
	mq := &models.Query{
		Expressions: make([]*models.Expr, 0, len(q.Expressions)),
	}
	for _, e := range q.Expressions {
		me := &models.Expr{Not: e.Not}
		switch v := e.Value.(type) {
		case *threading.Expr_Flag_:
			switch v.Flag {
			case threading.EXPR_FLAG_UNREAD:
				me.Value = &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD}
			case threading.EXPR_FLAG_UNREAD_REFERENCE:
				me.Value = &models.Expr_Flag_{Flag: models.EXPR_FLAG_UNREAD_REFERENCE}
			case threading.EXPR_FLAG_FOLLOWING:
				me.Value = &models.Expr_Flag_{Flag: models.EXPR_FLAG_FOLLOWING}
			default:
				return nil, errors.Errorf("unknown query flag type %s", v.Flag)
			}
		case *threading.Expr_ThreadType_:
			switch v.ThreadType {
			case threading.EXPR_THREAD_TYPE_PATIENT:
				me.Value = &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_PATIENT}
			case threading.EXPR_THREAD_TYPE_TEAM:
				me.Value = &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_TEAM}
			case threading.EXPR_THREAD_TYPE_SUPPORT:
				me.Value = &models.Expr_ThreadType_{ThreadType: models.EXPR_THREAD_TYPE_SUPPORT}
			default:
				return nil, errors.Errorf("unknown query thread type %s", v.ThreadType)
			}
		case *threading.Expr_Token:
			me.Value = &models.Expr_Token{Token: v.Token}
		default:
			return nil, errors.Errorf("unknown query expression type %T", e.Value)
		}
		mq.Expressions = append(mq.Expressions, me)
	}
	return mq, nil
}

func transformQueryToResponse(q *models.Query) (*threading.Query, error) {
	mq := &threading.Query{
		Expressions: make([]*threading.Expr, 0, len(q.Expressions)),
	}
	for _, e := range q.Expressions {
		me := &threading.Expr{Not: e.Not}
		switch v := e.Value.(type) {
		case *models.Expr_Flag_:
			switch v.Flag {
			case models.EXPR_FLAG_UNREAD:
				me.Value = &threading.Expr_Flag_{Flag: threading.EXPR_FLAG_UNREAD}
			case models.EXPR_FLAG_UNREAD_REFERENCE:
				me.Value = &threading.Expr_Flag_{Flag: threading.EXPR_FLAG_UNREAD_REFERENCE}
			case models.EXPR_FLAG_FOLLOWING:
				me.Value = &threading.Expr_Flag_{Flag: threading.EXPR_FLAG_FOLLOWING}
			default:
				return nil, errors.Errorf("unknown query flag type %s", v.Flag)
			}
		case *models.Expr_ThreadType_:
			switch v.ThreadType {
			case models.EXPR_THREAD_TYPE_PATIENT:
				me.Value = &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_PATIENT}
			case models.EXPR_THREAD_TYPE_TEAM:
				me.Value = &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_TEAM}
			case models.EXPR_THREAD_TYPE_SUPPORT:
				me.Value = &threading.Expr_ThreadType_{ThreadType: threading.EXPR_THREAD_TYPE_SUPPORT}
			default:
				return nil, errors.Errorf("unknown query thread type %s", v.ThreadType)
			}
		case *models.Expr_Token:
			me.Value = &threading.Expr_Token{Token: v.Token}
		default:
			return nil, errors.Errorf("unknown query expression type %T", e.Value)
		}
		mq.Expressions = append(mq.Expressions, me)
	}
	return mq, nil
}

func transformEndpointFromRequest(e *threading.Endpoint) (*models.Endpoint, error) {
	switch e.Channel {
	case threading.ENDPOINT_CHANNEL_APP:
		// TODO: remove this once it's not in the proto anymore
		return &models.Endpoint{Channel: models.ENDPOINT_CHANNEL_APP, ID: e.ID}, nil
	case threading.ENDPOINT_CHANNEL_EMAIL:
		return &models.Endpoint{Channel: models.ENDPOINT_CHANNEL_EMAIL, ID: e.ID}, nil
	case threading.ENDPOINT_CHANNEL_SMS:
		return &models.Endpoint{Channel: models.ENDPOINT_CHANNEL_SMS, ID: e.ID}, nil
	case threading.ENDPOINT_CHANNEL_VOICE:
		return &models.Endpoint{Channel: models.ENDPOINT_CHANNEL_VOICE, ID: e.ID}, nil
	}
	return nil, fmt.Errorf("Unknown endpoint channel %s", e.Channel.String())
}

func transformEndpointToResponse(e *models.Endpoint) (*threading.Endpoint, error) {
	switch e.Channel {
	case models.ENDPOINT_CHANNEL_APP:
		// TODO: remove this once it's not in the proto anymore
		return &threading.Endpoint{Channel: threading.ENDPOINT_CHANNEL_APP, ID: e.ID}, nil
	case models.ENDPOINT_CHANNEL_EMAIL:
		return &threading.Endpoint{Channel: threading.ENDPOINT_CHANNEL_EMAIL, ID: e.ID}, nil
	case models.ENDPOINT_CHANNEL_SMS:
		return &threading.Endpoint{Channel: threading.ENDPOINT_CHANNEL_SMS, ID: e.ID}, nil
	case models.ENDPOINT_CHANNEL_VOICE:
		return &threading.Endpoint{Channel: threading.ENDPOINT_CHANNEL_VOICE, ID: e.ID}, nil
	}
	return nil, fmt.Errorf("Unknown endpoint channel %s", e.Channel.String())
}

func transformThreadToResponse(thread *models.Thread, forExternal bool) (*threading.Thread, error) {
	t := &threading.Thread{
		ID:                   thread.ID.String(),
		OrganizationID:       thread.OrganizationID,
		PrimaryEntityID:      thread.PrimaryEntityID,
		LastMessageTimestamp: uint64(thread.LastMessageTimestamp.Unix()),
		LastMessageSummary:   thread.LastMessageSummary,
		CreatedTimestamp:     uint64(thread.Created.Unix()),
		MessageCount:         int32(thread.MessageCount),
		SystemTitle:          thread.SystemTitle,
		UserTitle:            thread.UserTitle,
	}
	var err error
	t.Type, err = transformThreadTypeToResponse(thread.Type)
	if err != nil {
		return nil, errors.Trace(err)
	}
	t.Origin, err = transformThreadOriginToResponse(thread.Origin)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(thread.LastPrimaryEntityEndpoints.Endpoints) != 0 {
		t.LastPrimaryEntityEndpoints = make([]*threading.Endpoint, len(thread.LastPrimaryEntityEndpoints.Endpoints))
		for i, ep := range thread.LastPrimaryEntityEndpoints.Endpoints {
			tc, err := transformEndpointChannelToResponse(ep.Channel)
			if err != nil {
				return nil, errors.Trace(err)
			}
			t.LastPrimaryEntityEndpoints[i] = &threading.Endpoint{
				Channel: tc,
				ID:      ep.ID,
			}
		}
	}
	if forExternal {
		t.LastMessageTimestamp = uint64(thread.LastExternalMessageTimestamp.Unix())
		t.LastMessageSummary = thread.LastExternalMessageSummary
	}
	return t, nil
}

func transformThreadTypeToResponse(tt models.ThreadType) (threading.ThreadType, error) {
	switch tt {
	case models.ThreadTypeUnknown:
		return threading.THREAD_TYPE_INVALID, nil
	case models.ThreadTypeExternal:
		return threading.THREAD_TYPE_EXTERNAL, nil
	case models.ThreadTypeTeam:
		return threading.THREAD_TYPE_TEAM, nil
	case models.ThreadTypeSetup:
		return threading.THREAD_TYPE_SETUP, nil
	case models.ThreadTypeSupport:
		return threading.THREAD_TYPE_SUPPORT, nil
	case models.ThreadTypeLegacyTeam:
		return threading.THREAD_TYPE_LEGACY_TEAM, nil
	case models.ThreadTypeSecureExternal:
		return threading.THREAD_TYPE_SECURE_EXTERNAL, nil
	}
	return threading.THREAD_TYPE_INVALID, errors.Errorf("unknown thread type '%s'", tt)
}

func transformThreadTypeFromRequest(tt threading.ThreadType) (models.ThreadType, error) {
	// Don't support creating threads with unknown types. The UNKNOWN type is only for old pre-migrated threads.
	switch tt {
	case threading.THREAD_TYPE_EXTERNAL:
		return models.ThreadTypeExternal, nil
	case threading.THREAD_TYPE_TEAM:
		return models.ThreadTypeTeam, nil
	case threading.THREAD_TYPE_SETUP:
		return models.ThreadTypeSetup, nil
	case threading.THREAD_TYPE_SUPPORT:
		return models.ThreadTypeSupport, nil
	case threading.THREAD_TYPE_LEGACY_TEAM:
		return models.ThreadTypeLegacyTeam, nil
	case threading.THREAD_TYPE_SECURE_EXTERNAL:
		return models.ThreadTypeSecureExternal, nil
	}
	return models.ThreadTypeUnknown, errors.Errorf("unknown thread type '%s'", tt)
}

func transformThreadOriginFromRequest(to threading.ThreadOrigin) (models.ThreadOrigin, error) {
	switch to {
	case threading.THREAD_ORIGIN_UNKNOWN:
		return models.ThreadOriginUnknown, nil
	case threading.THREAD_ORIGIN_ORGANIZATION_CODE:
		return models.ThreadOriginOrganizationCode, nil
	case threading.THREAD_ORIGIN_PATIENT_INVITE:
		return models.ThreadOriginPatientInvite, nil
	case threading.THREAD_ORIGIN_SYNC:
		return models.ThreadOriginPatientSync, nil
	}
	return models.ThreadOriginUnknown, errors.Errorf("unknown thread origin '%s'", to)
}

func transformThreadOriginToResponse(to models.ThreadOrigin) (threading.ThreadOrigin, error) {
	switch to {
	case models.ThreadOriginUnknown:
		return threading.THREAD_ORIGIN_UNKNOWN, nil
	case models.ThreadOriginOrganizationCode:
		return threading.THREAD_ORIGIN_ORGANIZATION_CODE, nil
	case models.ThreadOriginPatientInvite:
		return threading.THREAD_ORIGIN_PATIENT_INVITE, nil
	case models.ThreadOriginPatientSync:
		return threading.THREAD_ORIGIN_SYNC, nil
	}
	return threading.THREAD_ORIGIN_UNKNOWN, errors.Errorf("unknown thread origin '%s'", to)
}

func transformRequestEndpointChannelToDAL(c threading.Endpoint_Channel) (models.Endpoint_Channel, error) {
	var dc models.Endpoint_Channel
	switch c {
	case threading.ENDPOINT_CHANNEL_APP:
		dc = models.ENDPOINT_CHANNEL_APP
	case threading.ENDPOINT_CHANNEL_EMAIL:
		dc = models.ENDPOINT_CHANNEL_EMAIL
	case threading.ENDPOINT_CHANNEL_SMS:
		dc = models.ENDPOINT_CHANNEL_SMS
	case threading.ENDPOINT_CHANNEL_VOICE:
		dc = models.ENDPOINT_CHANNEL_VOICE
	default:
		return 0, errors.Errorf("Unknown dal layer endpoint channel type: %v", c)
	}
	return dc, nil
}

func transformEndpointChannelToResponse(c models.Endpoint_Channel) (threading.Endpoint_Channel, error) {
	var tc threading.Endpoint_Channel
	switch c {
	case models.ENDPOINT_CHANNEL_APP:
		tc = threading.ENDPOINT_CHANNEL_APP
	case models.ENDPOINT_CHANNEL_EMAIL:
		tc = threading.ENDPOINT_CHANNEL_EMAIL
	case models.ENDPOINT_CHANNEL_SMS:
		tc = threading.ENDPOINT_CHANNEL_SMS
	case models.ENDPOINT_CHANNEL_VOICE:
		tc = threading.ENDPOINT_CHANNEL_VOICE
	default:
		return 0, errors.Errorf("Unknown grpc layer endpoint channel type: %v", c)
	}
	return tc, nil
}

func transformThreadItemToResponse(item *models.ThreadItem, orgID string) (*threading.ThreadItem, error) {
	it := &threading.ThreadItem{
		ID:             item.ID.String(),
		Timestamp:      uint64(item.Created.Unix()),
		ActorEntityID:  item.ActorEntityID,
		Internal:       item.Internal,
		ThreadID:       item.ThreadID.String(),
		OrganizationID: orgID,
	}
	switch item.Type {
	case models.ItemTypeMessage:
		m2, err := TransformMessageToResponse(item.Data.(*models.Message))
		if err != nil {
			return nil, errors.Trace(err)
		}
		it.Item = &threading.ThreadItem_Message{
			Message: m2,
		}
	default:
		return nil, errors.Errorf("unknown thread item type %s", item.Type)
	}
	return it, nil
}

func TransformMessageToResponse(m *models.Message) (*threading.Message, error) {
	m2 := &threading.Message{
		Title:           m.Title,
		Text:            m.Text,
		Summary:         m.Summary,
		EditedTimestamp: m.EditedTimestamp,
		EditorEntityID:  m.EditorEntityID,
	}
	switch m.Status {
	case models.MESSAGE_STATUS_NORMAL:
		m2.Status = threading.MESSAGE_STATUS_NORMAL
	case models.MESSAGE_STATUS_DELETED:
		m2.Status = threading.MESSAGE_STATUS_DELETED
	default:
		return nil, errors.Errorf("unknown message status %s", m.Status)
	}
	if m.Source != nil {
		var err error
		m2.Source, err = transformEndpointToResponse(m.Source)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	if len(m.TextRefs) != 0 {
		m2.TextRefs = make([]*threading.Reference, len(m.TextRefs))
		for i, r := range m.TextRefs {
			var err error
			m2.TextRefs[i], err = transformReferenceToResponse(r)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}
	for _, a := range m.Attachments {
		at := &threading.Attachment{
			Title:     a.Title,
			UserTitle: a.UserTitle,
			URL:       a.URL,
			ContentID: a.ContentID,
		}
		switch adata := a.Data.(type) {
		case *models.Attachment_Audio:
			data := adata.Audio
			var durationNS uint64
			if data.DeprecatedDurationInSeconds != 0 {
				durationNS = uint64(data.DeprecatedDurationInSeconds) * 1e9
			} else {
				durationNS = data.DurationNS
			}
			at.Data = &threading.Attachment_Audio{
				Audio: &threading.AudioAttachment{
					Mimetype:   data.Mimetype,
					MediaID:    data.MediaID,
					DurationNS: durationNS,
				},
			}
			if at.ContentID == "" {
				at.ContentID = data.MediaID
			}
		case *models.Attachment_Image:
			data := adata.Image
			at.Data = &threading.Attachment_Image{
				Image: &threading.ImageAttachment{
					Mimetype: data.Mimetype,
					MediaID:  data.MediaID,
					Width:    data.Width,
					Height:   data.Height,
				},
			}
			if at.ContentID == "" {
				at.ContentID = data.MediaID
			}
		case *models.Attachment_Generic:
			data := adata.Generic
			at.Data = &threading.Attachment_GenericURL{
				GenericURL: &threading.GenericURLAttachment{
					URL:      data.URL,
					Mimetype: data.Mimetype,
				},
			}
			if at.ContentID == "" {
				at.ContentID = data.URL
			}
		case *models.Attachment_Document:
			data := adata.Document
			at.Data = &threading.Attachment_Document{
				Document: &threading.DocumentAttachment{
					Mimetype: data.Mimetype,
					MediaID:  data.MediaID,
					Name:     data.Name,
				},
			}
			if at.ContentID == "" {
				at.ContentID = data.MediaID
			}
		case *models.Attachment_Visit:
			data := adata.Visit
			at.Data = &threading.Attachment_Visit{
				Visit: &threading.VisitAttachment{
					VisitID:   data.VisitID,
					VisitName: data.VisitName,
				},
			}
			if at.ContentID == "" {
				// TODO: this isn't right as it should be the source layout ID
				at.ContentID = data.VisitID
			}
		case *models.Attachment_Video:
			data := adata.Video
			at.Data = &threading.Attachment_Video{
				Video: &threading.VideoAttachment{
					Mimetype:   data.Mimetype,
					MediaID:    data.MediaID,
					DurationNS: data.DurationNS,
				},
			}
			if at.ContentID == "" {
				at.ContentID = data.MediaID
			}
		case *models.Attachment_CarePlan:
			data := adata.CarePlan
			at.Data = &threading.Attachment_CarePlan{
				CarePlan: &threading.CarePlanAttachment{
					CarePlanID:   data.CarePlanID,
					CarePlanName: data.CarePlanName,
				},
			}
			if at.ContentID == "" {
				at.ContentID = data.CarePlanID
			}
		case *models.Attachment_PaymentRequest:
			data := adata.PaymentRequest
			at.Data = &threading.Attachment_PaymentRequest{
				PaymentRequest: &threading.PaymentRequestAttachment{
					PaymentID: data.PaymentID,
				},
			}
			if at.ContentID == "" {
				at.ContentID = data.PaymentID
			}
		default:
			return nil, errors.Errorf("invalid attachment type %T", a.Data)

		}
		m2.Attachments = append(m2.Attachments, at)
	}
	if len(m.Destinations) != 0 {
		m2.Destinations = make([]*threading.Endpoint, len(m.Destinations))
		for i, dc := range m.Destinations {
			e, err := transformEndpointToResponse(dc)
			if err != nil {
				return nil, errors.Trace(err)
			}
			m2.Destinations[i] = e
		}
	}
	return m2, nil
}

func transformReferenceToResponse(r *models.Reference) (*threading.Reference, error) {
	tr := &threading.Reference{
		ID: r.ID,
	}
	switch r.Type {
	case models.REFERENCE_TYPE_ENTITY:
		tr.Type = threading.REFERENCE_TYPE_ENTITY
	default:
		return nil, errors.Errorf("unknown reference type %s", r.Type.String())
	}
	return tr, nil
}

func transformSavedQueryToResponse(sq *models.SavedQuery) (*threading.SavedQuery, error) {
	query, err := transformQueryToResponse(sq.Query)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var t threading.SavedQueryType
	switch sq.Type {
	case models.SavedQueryTypeNormal:
		t = threading.SAVED_QUERY_TYPE_NORMAL
	case models.SavedQueryTypeNotifications:
		t = threading.SAVED_QUERY_TYPE_NOTIFICATIONS
	default:
		return nil, errors.Errorf("unknown saved query type %s", sq.Type)
	}
	return &threading.SavedQuery{
		ID:                   sq.ID.String(),
		Type:                 t,
		Ordinal:              int32(sq.Ordinal),
		Query:                query,
		Title:                sq.Title,
		Unread:               uint32(sq.Unread),
		Total:                uint32(sq.Total),
		EntityID:             sq.EntityID,
		Hidden:               sq.Hidden,
		NotificationsEnabled: sq.NotificationsEnabled,
	}, nil
}

// From request

func transformAttachmentsFromRequest(atts []*threading.Attachment) ([]*models.Attachment, error) {
	if len(atts) == 0 {
		return nil, nil
	}
	as := make([]*models.Attachment, 0, len(atts))
	for _, a := range atts {
		at := &models.Attachment{
			Title:     a.Title,
			UserTitle: a.UserTitle,
			URL:       a.URL,
			ContentID: a.ContentID,
		}
		switch adata := a.Data.(type) {
		case *threading.Attachment_Audio:
			data := adata.Audio
			at.Data = &models.Attachment_Audio{
				Audio: &models.AudioAttachment{
					Mimetype:   data.Mimetype,
					MediaID:    data.MediaID,
					DurationNS: data.DurationNS,
				},
			}
		case *threading.Attachment_CarePlan:
			data := adata.CarePlan
			at.Data = &models.Attachment_CarePlan{
				CarePlan: &models.CarePlanAttachment{
					CarePlanName: data.CarePlanName,
					CarePlanID:   data.CarePlanID,
				},
			}
		case *threading.Attachment_GenericURL:
			data := adata.GenericURL
			at.Data = &models.Attachment_Generic{
				Generic: &models.GenericAttachment{
					URL:      data.URL,
					Mimetype: data.Mimetype,
				},
			}
		case *threading.Attachment_Image:
			data := adata.Image
			at.Data = &models.Attachment_Image{
				Image: &models.ImageAttachment{
					Mimetype: data.Mimetype,
					MediaID:  data.MediaID,
					Width:    data.Width,
					Height:   data.Height,
				},
			}
		case *threading.Attachment_Video:
			data := adata.Video
			at.Data = &models.Attachment_Video{
				Video: &models.VideoAttachment{
					Mimetype:   data.Mimetype,
					MediaID:    data.MediaID,
					DurationNS: data.DurationNS,
				},
			}
		case *threading.Attachment_Document:
			data := adata.Document
			at.Data = &models.Attachment_Document{
				Document: &models.DocumentAttachment{
					Mimetype: data.Mimetype,
					MediaID:  data.MediaID,
					Name:     data.Name,
				},
			}
		case *threading.Attachment_Visit:
			data := adata.Visit
			at.Data = &models.Attachment_Visit{
				Visit: &models.VisitAttachment{
					VisitName: data.VisitName,
					VisitID:   data.VisitID,
				},
			}
		case *threading.Attachment_PaymentRequest:
			data := adata.PaymentRequest
			at.Data = &models.Attachment_PaymentRequest{
				PaymentRequest: &models.PaymentRequestAttachment{
					PaymentID: data.PaymentID,
				},
			}
		default:
			return nil, errors.Errorf("invalid attachment type %T", a.Data)

		}
		as = append(as, at)
	}
	return as, nil
}

func transformScheduledMessagesToResponse(sms []*models.ScheduledMessage) ([]*threading.ScheduledMessage, error) {
	rsms := make([]*threading.ScheduledMessage, len(sms))
	for i, sm := range sms {
		rsm, err := transformScheduledMessageToResponse(sm)
		if err != nil {
			return nil, errors.Trace(err)
		}
		rsms[i] = rsm
	}
	return rsms, nil
}

func transformScheduledMessageToResponse(sm *models.ScheduledMessage) (*threading.ScheduledMessage, error) {
	var sentAt uint64
	if sm.SentAt != nil {
		sentAt = uint64(sm.SentAt.Unix())
	}

	var sentThreadItemID string
	if sm.SentThreadItemID != nil {
		sentThreadItemID = sm.SentThreadItemID.String()
	}
	rsm := &threading.ScheduledMessage{
		ID:               sm.ID.String(),
		ScheduledFor:     uint64(sm.ScheduledFor.Unix()),
		ActorEntityID:    sm.ActorEntityID,
		Internal:         sm.Internal,
		ThreadID:         sm.ThreadID.String(),
		SentAt:           sentAt,
		SentThreadItemID: sentThreadItemID,
	}
	switch sm.Type {
	case models.ItemTypeMessage:
		msg, err := TransformMessageToResponse(sm.Data.(*models.Message))
		if err != nil {
			return nil, errors.Trace(err)
		}
		rsm.Content = &threading.ScheduledMessage_Message{
			Message: msg,
		}
	default:
		return nil, errors.Errorf("Unknown scheduled message type %s", sm.Type)
	}
	return rsm, nil
}

func transformScheduledMessageStatusFromRequest(status threading.ScheduledMessageStatus) (models.ScheduledMessageStatus, error) {
	switch status {
	case threading.SCHEDULED_MESSAGE_STATUS_PENDING:
		return models.ScheduledMessageStatusPending, nil
	case threading.SCHEDULED_MESSAGE_STATUS_SENT:
		return models.ScheduledMessageStatusSent, nil
	case threading.SCHEDULED_MESSAGE_STATUS_DELETED:
		return models.ScheduledMessageStatusDeleted, nil

	}
	return "", errors.Errorf("Unknown scheduled message status %s", status)
}
