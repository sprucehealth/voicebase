package main

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

func transformAccountToResponse(a *auth.Account) (*account, error) {
	return &account{
		ID: a.ID,
	}, nil
}

func threadTitleForEntity(e *directory.Entity) string {
	if e.Info.DisplayName != "" {
		return e.Info.DisplayName
	}
	for _, c := range e.Contacts {
		if c.ContactType == directory.ContactType_PHONE {
			pn, err := phone.Format(c.Value, phone.Pretty)
			if err != nil {
				return c.Value
			}
			return pn
		}
		return c.Value
	}
	// TODO: not sure what to use when there's no name or contacts
	return e.ID
}

func transformContactsToResponse(contacts []*directory.Contact) ([]*contactInfo, error) {
	cs := make([]*contactInfo, len(contacts))
	for i, c := range contacts {

		ci := &contactInfo{
			ID:           c.ID,
			Value:        c.Value,
			DisplayValue: c.Value,
			Provisioned:  c.Provisioned,
			Label:        c.Label,
		}
		switch c.ContactType {
		case directory.ContactType_EMAIL:
			ci.Type = contactTypeEmail
		case directory.ContactType_PHONE:
			ci.Type = contactTypePhone
			pn, err := phone.Format(c.Value, phone.Pretty)
			if err == nil {
				ci.DisplayValue = pn
			}
		default:
			return nil, errors.Trace(fmt.Errorf("unsupported contact type %s", c.ContactType.String()))
		}
		cs[i] = ci
	}
	return cs, nil
}

func transformThreadToResponse(t *threading.Thread) (*thread, error) {
	th := &thread{
		ID:                   t.ID,
		OrganizationID:       t.OrganizationID,
		PrimaryEntityID:      t.PrimaryEntityID,
		Subtitle:             t.LastMessageSummary,
		LastMessageTimestamp: t.LastMessageTimestamp,
		Unread:               t.Unread,
	}
	for i, ep := range t.LastPrimaryEntityEndpoints {
		th.LastPrimaryEntityEndpoints[i] = &endpoint{
			Channel: ep.Channel.String(),
			ID:      ep.ID,
		}
	}
	return th, nil
}

func transformThreadItemToResponse(item *threading.ThreadItem, uuid, accountID string, mediaSigner *media.Signer) (*threadItem, error) {
	it := &threadItem{
		ID:             item.ID,
		UUID:           uuid,
		Timestamp:      item.Timestamp,
		ActorEntityID:  item.ActorEntityID,
		Internal:       item.Internal,
		ThreadID:       item.ThreadID,
		OrganizationID: item.OrganizationID,
	}
	switch item.Type {
	case threading.ThreadItem_MESSAGE:
		m := item.GetMessage()
		m2 := &message{
			ThreadItemID:  item.ID,
			SummaryMarkup: m.Title,
			TextMarkup:    m.Text,
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
				// TODO: Twilio seems to round up the duration which causes problems with the progress bar in the app
				//       so reduce the duration by half a second to try to account for that. The real fix of actually
				//       processing the mp3 to figure out the accurate duration should be done when there's time.
				duration := float64(d.DurationInSeconds) - 0.5
				data = &audioAttachment{
					Mimetype:          d.Mimetype,
					URL:               signedURL,
					DurationInSeconds: duration,
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
		IsEditable:    e.Type != directory.EntityType_SYSTEM,
		Contacts:      oc,
		FirstName:     e.Info.FirstName,
		MiddleInitial: e.Info.MiddleInitial,
		LastName:      e.Info.LastName,
		GroupName:     e.Info.GroupName,
		DisplayName:   e.Info.DisplayName,
		ShortTitle:    e.Info.ShortTitle,
		LongTitle:     e.Info.LongTitle,
		Note:          e.Info.Note,
	}, nil
}

func transformOrganizationToResponse(org *directory.Entity, provider *directory.Entity) (*organization, error) {
	o := &organization{
		ID:   org.ID,
		Name: org.Info.DisplayName,
	}

	oc, err := transformContactsToResponse(org.Contacts)
	if err != nil {
		return nil, fmt.Errorf("failed to transform entity contacts: %+v", err)
	}

	o.Contacts = oc

	e, err := transformEntityToResponse(provider)
	if err != nil {
		return nil, err
	}
	o.Entity = e

	return o, nil
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

func transformStringListSettingToResponse(config *settings.Config, value *settings.Value) *stringListSetting {
	return &stringListSetting{
		Key:         config.Key,
		Subkey:      value.Key.Subkey,
		Title:       config.Title,
		Description: config.Description,
		Value: &stringListSettingValue{
			Values: value.GetStringList().Values,
		},
	}
}

func transformBooleanSettingToResponse(config *settings.Config, value *settings.Value) *booleanSetting {
	return &booleanSetting{
		Key:         config.Key,
		Subkey:      value.Key.Subkey,
		Title:       config.Title,
		Description: config.Description,
		Value: &booleanSettingValue{
			Value: value.GetBoolean().Value,
		},
	}
}

func transformMultiSelectToResponse(config *settings.Config, value *settings.Value) *selectSetting {
	ss := &selectSetting{
		Key:         config.Key,
		Subkey:      value.Key.Subkey,
		Title:       config.Title,
		Description: config.Description,
	}

	var items []*settings.Item
	var values []*settings.ItemValue
	if config.Type == settings.ConfigType_SINGLE_SELECT {
		items = config.GetSingleSelect().Items
		if value.GetSingleSelect().Item != nil {
			values = []*settings.ItemValue{value.GetSingleSelect().Item}
		}
	} else {
		items = config.GetMultiSelect().Items
		values = value.GetMultiSelect().Items
	}

	ss.Options = make([]*selectableItem, len(items))
	ss.Value = &selectableSettingValue{
		Items: make([]*selectableItemValue, len(values)),
	}

	for i, option := range items {
		ss.Options[i] = &selectableItem{
			ID:            option.ID,
			Label:         option.Label,
			AllowFreeText: option.AllowFreeText,
		}
	}

	for i, v := range values {
		ss.Value.Items[i] = &selectableItemValue{
			ID:   v.ID,
			Text: v.FreeTextResponse,
		}
	}

	return ss
}

func transformEntityContactToEndpoint(c *directory.Contact) (*endpoint, error) {
	var channel string
	var displayValue string
	var err error
	switch c.ContactType {
	case directory.ContactType_EMAIL:
		channel = endpointChannelEmail
		displayValue = c.Value
	case directory.ContactType_PHONE:
		channel = endpointChannelSMS
		displayValue, err = phone.Format(c.Value, phone.Pretty)
		if err != nil {
			return nil, errors.Trace(err)
		}
	default:
		return nil, fmt.Errorf("unknown contact type %v", c.ContactType)
	}
	return &endpoint{
		Channel:      channel,
		ID:           c.Value,
		DisplayValue: displayValue,
	}, nil
}
