package main

import (
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

func transformContactsToResponse(contacts []*directory.Contact) ([]*contactInfo, error) {
	cs := make([]*contactInfo, len(contacts))
	for i, c := range contacts {
		ci := &contactInfo{
			Value:       c.Value,
			Provisioned: c.Provisioned,
		}
		switch c.ContactType {
		case directory.ContactType_EMAIL:
			ci.Type = contactTypeEmail
		case directory.ContactType_PHONE:
			ci.Type = contactTypePhone
		default:
			return nil, errors.Trace(fmt.Errorf("unsupported contact type %s", c.ContactType.String()))
		}
		cs[i] = ci
	}
	return cs, nil
}

func transformThreadToResponse(t *threading.Thread) (*thread, error) {
	return &thread{
		ID:              t.ID,
		OrganizationID:  t.OrganizationID,
		PrimaryEntityID: t.PrimaryEntityID,
	}, nil
}

func transformThreadItemToResponse(item *threading.ThreadItem) (*threadItem, error) {
	it := &threadItem{
		ID:            item.ID,
		Timestamp:     item.Timestamp,
		ActorEntityID: item.ActorEntityID,
		Internal:      item.Internal,
	}
	switch item.Type {
	case threading.ThreadItem_MESSAGE:
		m := item.GetMessage()
		m2 := &message{
			Text:   m.Text,
			Status: m.Status.String(),
			Source: &endpoint{
				Channel: m.Source.Channel.String(),
				ID:      m.Source.ID,
			},
			// TODO: EditorEntityID
			// TODO: EditedTimestamp
		}
		for _, a := range m.Attachments {
			var data interface{}
			switch a.Type {
			case threading.Attachment_AUDIO:
				d := a.GetAudio()
				if d.Mimetype == "" { // TODO
					d.Mimetype = "audio/mp3"
				}
				data = &audioAttachment{
					Mimetype:          d.Mimetype,
					URL:               d.URL,
					DurationInSeconds: int(d.DurationInSeconds),
				}
				// TODO
				if a.Title == "" {
					a.Title = "Audio"
				}
				if a.URL == "" {
					a.URL = d.URL
				}
			case threading.Attachment_IMAGE:
				d := a.GetImage()
				if d.Mimetype == "" { // TODO
					d.Mimetype = "image/jpeg"
				}
				data = &imageAttachment{
					Mimetype: d.Mimetype,
					URL:      d.URL,
					Width:    int(d.Width),
					Height:   int(d.Height),
				}
				// TODO
				if a.Title == "" {
					a.Title = "Photo"
				}
				if a.URL == "" {
					a.URL = d.URL
				}
			default:
				return nil, errors.Trace(fmt.Errorf("unknown attachment type %s", a.Type.String()))
			}
			m2.Attachments = append(m2.Attachments, &attachment{
				Title: a.Title,
				URL:   a.URL,
				Data:  data,
			})
		}
		for _, dc := range m.Destinations {
			m2.Destinations = append(m2.Destinations, &endpoint{
				Channel: dc.Channel.String(),
				ID:      dc.ID,
			})
		}
		it.Data = m2
	default:
		return nil, errors.Trace(fmt.Errorf("unknown thread item type %s", item.Type.String()))
	}
	return it, nil
}

func transformSavedQueryToResponse(sq *threading.SavedQuery) (*savedThreadQuery, error) {
	return &savedThreadQuery{
		ID:             sq.ID,
		OrganizationID: sq.OrganizationID,
	}, nil
}

func transformEntityToResponse(e *directory.Entity) (*entity, error) {
	oc, err := transformContactsToResponse(e.Contacts)
	if err != nil {
		return nil, errors.Trace(fmt.Errorf("failed to transform contacts for entity %s: %s", e.ID, err))
	}
	return &entity{
		ID:       e.ID,
		Name:     e.Name,
		Contacts: oc,
	}, nil
}
