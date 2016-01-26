package main

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

func threadTitleForEntity(e *directory.Entity) string {
	if e.Info.DisplayName != "" {
		return e.Info.DisplayName
	}
	for _, c := range e.Contacts {
		return c.Value
	}
	// TODO: not sure what to use when there's no name or contacts
	return e.ID
}

func transformContactsToResponse(contacts []*directory.Contact) ([]*contactInfo, error) {
	cs := make([]*contactInfo, len(contacts))
	for i, c := range contacts {
		ci := &contactInfo{
			ID:          c.ID,
			Value:       c.Value,
			Provisioned: c.Provisioned,
			Label:       c.Label,
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
		ID:                   t.ID,
		OrganizationID:       t.OrganizationID,
		PrimaryEntityID:      t.PrimaryEntityID,
		Subtitle:             t.LastMessageSummary,
		LastMessageTimestamp: t.LastMessageTimestamp,
		Unread:               t.Unread,
	}, nil
}

func transformThreadItemToResponse(item *threading.ThreadItem, uuid, accountID string, mediaSigner *media.Signer) (*threadItem, error) {
	it := &threadItem{
		ID:            item.ID,
		UUID:          uuid,
		Timestamp:     item.Timestamp,
		ActorEntityID: item.ActorEntityID,
		Internal:      item.Internal,
	}
	switch item.Type {
	case threading.ThreadItem_MESSAGE:
		m := item.GetMessage()
		m2 := &message{
			ThreadItemID: item.ID,
			Title:        m.Title,
			Text:         m.Text,
			Status:       m.Status.String(),
			Source: &endpoint{
				Channel: m.Source.Channel.String(),
				ID:      m.Source.ID,
			},
			// TODO: EditorEntityID
			// TODO: EditedTimestamp
		}
		for _, r := range m.TextRefs {
			m2.Refs = append(m2.Refs, &reference{
				ID:   r.ID,
				Type: strings.ToLower(r.Type.String()),
			})
		}
		for _, a := range m.Attachments {
			var data interface{}
			switch a.Type {
			case threading.Attachment_AUDIO:
				d := a.GetAudio()
				if d.Mimetype == "" { // TODO
					d.Mimetype = "audio/mp3"
				}

				mediaID, err := media.ParseMediaID(d.URL)
				if err != nil {
					golog.Errorf("Unable to parse mediaID out of url %s", d.URL)
				}

				signedURL, err := mediaSigner.SignedURL(mediaID, d.Mimetype, accountID, 0, 0, false)
				if err != nil {
					return nil, err
				}
				data = &audioAttachment{
					Mimetype:          d.Mimetype,
					URL:               signedURL,
					DurationInSeconds: float64(d.DurationInSeconds),
				}
				// TODO
				if a.Title == "" {
					a.Title = "Audio"
				}
				if a.URL == "" {
					a.URL = signedURL
				}
			case threading.Attachment_IMAGE:
				d := a.GetImage()
				if d.Mimetype == "" { // TODO
					d.Mimetype = "image/jpeg"
				}
				data = &imageAttachment{
					Mimetype: d.Mimetype,
					URL:      d.URL,
				}
				// TODO
				if a.Title == "" {
					a.Title = "Photo"
				}

				mediaID, err := media.ParseMediaID(d.URL)
				if err != nil {
					golog.Errorf("Unable to parse mediaID out of url %s", d.URL)
				}

				signedURL, err := mediaSigner.SignedURL(mediaID, d.Mimetype, accountID, 0, 0, false)
				if err != nil {
					return nil, err
				}

				if a.URL == "" {
					a.URL = signedURL
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
		ID:            e.ID,
		Contacts:      oc,
		FirstName:     e.Info.FirstName,
		MiddleInitial: e.Info.MiddleInitial,
		LastName:      e.Info.LastName,
		GroupName:     e.Info.GroupName,
		DisplayName:   e.Info.DisplayName,
		Note:          e.Info.Note,
	}, nil
}

func transformThreadItemViewDetailsToResponse(tivds []*threading.ThreadItemViewDetails) ([]*threadItemViewDetails, error) {
	rivds := make([]*threadItemViewDetails, len(tivds))
	for i, tivd := range tivds {
		rivds[i] = &threadItemViewDetails{
			ThreadItemID:  tivd.ThreadItemID,
			ActorEntityID: tivd.EntityID,
			ViewTime:      tivd.ViewTime,
		}
	}
	return rivds, nil
}
