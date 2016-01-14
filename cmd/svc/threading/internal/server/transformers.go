package server

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/threading/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/threading"
)

func transformThreadToResponse(thread *models.Thread, forExternal bool) (*threading.Thread, error) {
	t := &threading.Thread{
		ID:                   thread.ID.String(),
		OrganizationID:       thread.OrganizationID,
		PrimaryEntityID:      thread.PrimaryEntityID,
		LastMessageTimestamp: uint64(thread.LastMessageTimestamp.Unix()),
		LastMessageSummary:   thread.LastMessageSummary,
	}
	if forExternal {
		t.LastMessageTimestamp = uint64(thread.LastExternalMessageTimestamp.Unix())
		t.LastMessageSummary = thread.LastExternalMessageSummary
	}
	return t, nil
}

func transformThreadItemToResponse(item *models.ThreadItem) (*threading.ThreadItem, error) {
	it := &threading.ThreadItem{
		ID:            item.ID.String(),
		Timestamp:     uint64(item.Created.Unix()),
		ActorEntityID: item.ActorEntityID,
		Internal:      item.Internal,
		Type:          threading.ThreadItem_Type(threading.ThreadItem_Type_value[string(item.Type)]), // TODO
	}
	switch item.Type {
	case models.ItemTypeMessage:
		m := item.Data.(*models.Message)
		m2 := &threading.Message{
			Title:  m.Title,
			Text:   m.Text,
			Status: threading.Message_Status(threading.Message_Status_value[m.Status.String()]), // TODO
			Source: &threading.Endpoint{
				Channel: threading.Endpoint_Channel(threading.Endpoint_Channel_value[m.Source.Channel.String()]), // TODO
				ID:      m.Source.ID,
			},
			EditedTimestamp: m.EditedTimestamp,
			EditorEntityID:  m.EditorEntityID,
			TextRefs:        make([]*threading.Reference, len(m.TextRefs)),
		}
		// TODO: this is temporary since old messages don't have a title
		if m2.Title == "" {
			m2.Title = m2.Source.ID
		}
		for i, r := range m.TextRefs {
			var err error
			m2.TextRefs[i], err = transformReferenceToResponse(r)
			if err != nil {
				return nil, errors.Trace(err)
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
				at.Data = &threading.Attachment_Audio{
					Audio: &threading.AudioAttachment{
						Mimetype:          data.Mimetype,
						URL:               data.URL,
						DurationInSeconds: data.DurationInSeconds,
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
			default:
				return nil, errors.New("invalid attachment type " + a.Type.String())

			}
			m2.Attachments = append(m2.Attachments, at)
		}
		for _, dc := range m.Destinations {
			m2.Destinations = append(m2.Destinations, &threading.Endpoint{
				Channel: threading.Endpoint_Channel(threading.Endpoint_Channel_value[dc.Channel.String()]), // TODO
				ID:      dc.ID,
			})
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
					Mimetype:          data.Mimetype,
					URL:               data.URL,
					DurationInSeconds: data.DurationInSeconds,
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
		default:
			return nil, errors.New("invalid attachment type " + a.Type.String())

		}
		as = append(as, at)
	}
	return as, nil
}
