package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/threading"
)

func transformEndpointFromRequest(e *threading.Endpoint) (*models.Endpoint, error) {
	switch e.Channel {
	case threading.Endpoint_APP:
		// TODO: remove this once it's not in the proto anymore
		return &models.Endpoint{Channel: models.Endpoint_APP, ID: e.ID}, nil
	case threading.Endpoint_EMAIL:
		return &models.Endpoint{Channel: models.Endpoint_EMAIL, ID: e.ID}, nil
	case threading.Endpoint_SMS:
		return &models.Endpoint{Channel: models.Endpoint_SMS, ID: e.ID}, nil
	case threading.Endpoint_VOICE:
		return &models.Endpoint{Channel: models.Endpoint_VOICE, ID: e.ID}, nil
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
		return threading.ThreadType_UNKNOWN, nil
	case models.ThreadTypeExternal:
		return threading.ThreadType_EXTERNAL, nil
	case models.ThreadTypeTeam:
		return threading.ThreadType_TEAM, nil
	case models.ThreadTypeSetup:
		return threading.ThreadType_SETUP, nil
	case models.ThreadTypeSupport:
		return threading.ThreadType_SUPPORT, nil
	}
	return threading.ThreadType_UNKNOWN, errors.Trace(fmt.Errorf("unknown thread type '%s'", tt))
}

func transformThreadTypeFromRequest(tt threading.ThreadType) (models.ThreadType, error) {
	// Don't support creating threads with unknown types. The UNKNOWN type is only for old pre-migrated threads.
	switch tt {
	case threading.ThreadType_EXTERNAL:
		return models.ThreadTypeExternal, nil
	case threading.ThreadType_TEAM:
		return models.ThreadTypeTeam, nil
	case threading.ThreadType_SETUP:
		return models.ThreadTypeSetup, nil
	case threading.ThreadType_SUPPORT:
		return models.ThreadTypeSupport, nil
	}
	return models.ThreadTypeUnknown, errors.Trace(fmt.Errorf("unknown thread type '%s'", tt))
}

func transformRequestEndpointChannelToDAL(c threading.Endpoint_Channel) (models.Endpoint_Channel, error) {
	var dc models.Endpoint_Channel
	switch c {
	case threading.Endpoint_APP:
		dc = models.Endpoint_APP
	case threading.Endpoint_EMAIL:
		dc = models.Endpoint_EMAIL
	case threading.Endpoint_SMS:
		dc = models.Endpoint_SMS
	case threading.Endpoint_VOICE:
		dc = models.Endpoint_VOICE
	default:
		return 0, errors.Trace(fmt.Errorf("Unknown dal layer endpoint channel type: %v", c))
	}
	return dc, nil
}

func transformEndpointChannelToResponse(c models.Endpoint_Channel) (threading.Endpoint_Channel, error) {
	var tc threading.Endpoint_Channel
	switch c {
	case models.Endpoint_APP:
		tc = threading.Endpoint_APP
	case models.Endpoint_EMAIL:
		tc = threading.Endpoint_EMAIL
	case models.Endpoint_SMS:
		tc = threading.Endpoint_SMS
	case models.Endpoint_VOICE:
		tc = threading.Endpoint_VOICE
	default:
		return 0, errors.Trace(fmt.Errorf("Unknown grpc layer endpoint channel type: %v", c))
	}
	return tc, nil
}

func transformThreadItemToResponse(item *models.ThreadItem, orgID string) (*threading.ThreadItem, error) {
	it := &threading.ThreadItem{
		ID:             item.ID.String(),
		Timestamp:      uint64(item.Created.Unix()),
		ActorEntityID:  item.ActorEntityID,
		Internal:       item.Internal,
		Type:           threading.ThreadItem_Type(threading.ThreadItem_Type_value[string(item.Type)]), // TODO
		ThreadID:       item.ThreadID.String(),
		OrganizationID: orgID,
	}
	switch item.Type {
	case models.ItemTypeMessage:
		m := item.Data.(*models.Message)
		m2 := &threading.Message{
			Title:           m.Title,
			Text:            m.Text,
			Status:          threading.Message_Status(threading.Message_Status_value[m.Status.String()]), // TODO
			Summary:         m.Summary,
			EditedTimestamp: m.EditedTimestamp,
			EditorEntityID:  m.EditorEntityID,
		}
		if m.Source != nil {
			m2.Source = &threading.Endpoint{
				Channel: threading.Endpoint_Channel(threading.Endpoint_Channel_value[m.Source.Channel.String()]), // TODO
				ID:      m.Source.ID,
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
				Title: a.Title,
				URL:   a.URL,
			}
			switch a.Type {
			case models.Attachment_AUDIO:
				data := a.GetAudio()
				at.Type = threading.Attachment_AUDIO
				var durationNS uint64
				if data.DeprecatedDurationInSeconds != 0 {
					durationNS = uint64(data.DeprecatedDurationInSeconds) * 1e9
				} else {
					durationNS = data.DurationNS
				}
				at.Data = &threading.Attachment_Audio{
					Audio: &threading.AudioAttachment{
						Mimetype:   data.Mimetype,
						URL:        data.URL,
						DurationNS: durationNS,
					},
				}
			case models.Attachment_IMAGE:
				data := a.GetImage()
				at.Type = threading.Attachment_IMAGE
				at.Data = &threading.Attachment_Image{
					Image: &threading.ImageAttachment{
						Mimetype: data.Mimetype,
						URL:      data.URL,
						Width:    data.Width,
						Height:   data.Height,
					},
				}
			case models.Attachment_GENERIC:
				data := a.GetGeneric()
				at.Type = threading.Attachment_GENERIC_URL
				at.Data = &threading.Attachment_GenericURL{
					GenericURL: &threading.GenericURLAttachment{
						URL:      data.URL,
						Mimetype: data.Mimetype,
					},
				}
			default:
				return nil, errors.New("invalid attachment type " + a.Type.String())

			}
			m2.Attachments = append(m2.Attachments, at)
		}
		if len(m.Destinations) != 0 {
			m2.Destinations = make([]*threading.Endpoint, len(m.Destinations))
			for i, dc := range m.Destinations {
				m2.Destinations[i] = &threading.Endpoint{
					Channel: threading.Endpoint_Channel(threading.Endpoint_Channel_value[dc.Channel.String()]), // TODO
					ID:      dc.ID,
				}
			}
		}
		it.Item = &threading.ThreadItem_Message{
			Message: m2,
		}
	default:
		return nil, errors.Trace(fmt.Errorf("unknown thread item type %s", item.Type))
	}
	return it, nil
}

func transformReferenceToResponse(r *models.Reference) (*threading.Reference, error) {
	tr := &threading.Reference{
		ID: r.ID,
	}
	switch r.Type {
	case models.Reference_ENTITY:
		tr.Type = threading.Reference_ENTITY
	default:
		return nil, errors.Trace(fmt.Errorf("unknown reference type %s", r.Type.String()))
	}
	return tr, nil
}

func transformSavedQueryToResponse(sq *models.SavedQuery) (*threading.SavedQuery, error) {
	return &threading.SavedQuery{
		ID:             sq.ID.String(),
		OrganizationID: sq.OrganizationID,
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
			Title: a.Title,
			URL:   a.URL,
		}
		switch a.Type {
		case threading.Attachment_AUDIO:
			data := a.GetAudio()
			at.Type = models.Attachment_AUDIO
			at.Data = &models.Attachment_Audio{
				Audio: &models.AudioAttachment{
					Mimetype:   data.Mimetype,
					URL:        data.URL,
					DurationNS: data.DurationNS,
				},
			}
		case threading.Attachment_IMAGE:
			data := a.GetImage()
			at.Type = models.Attachment_IMAGE
			at.Data = &models.Attachment_Image{
				Image: &models.ImageAttachment{
					Mimetype: data.Mimetype,
					URL:      data.URL,
					Width:    data.Width,
					Height:   data.Height,
				},
			}
		case threading.Attachment_GENERIC_URL:
			data := a.GetGenericURL()
			at.Type = models.Attachment_GENERIC
			at.Data = &models.Attachment_Generic{
				Generic: &models.GenericAttachment{
					URL:      data.URL,
					Mimetype: data.Mimetype,
				},
			}
		default:
			return nil, errors.New("invalid attachment type " + a.Type.String())

		}
		as = append(as, at)
	}
	return as, nil
}
