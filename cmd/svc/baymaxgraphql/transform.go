package main

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

func transformAccountToResponse(a *auth.Account) (*models.Account, error) {
	return &models.Account{
		ID: a.ID,
	}, nil
}

func threadTitleForEntity(e *directory.Entity) string {
	if e.Type == directory.EntityType_ORGANIZATION {
		return fmt.Sprintf("Team (%s)", e.Info.DisplayName)
	}
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

func transformContactsToResponse(contacts []*directory.Contact) ([]*models.ContactInfo, error) {
	cs := make([]*models.ContactInfo, len(contacts))
	for i, c := range contacts {

		ci := &models.ContactInfo{
			ID:           c.ID,
			Value:        c.Value,
			DisplayValue: c.Value,
			Provisioned:  c.Provisioned,
			Label:        c.Label,
		}
		switch c.ContactType {
		case directory.ContactType_EMAIL:
			ci.Type = models.ContactTypeEmail
		case directory.ContactType_PHONE:
			ci.Type = models.ContactTypePhone
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

func transformThreadToResponse(t *threading.Thread) (*models.Thread, error) {
	th := &models.Thread{
		ID:                         t.ID,
		OrganizationID:             t.OrganizationID,
		PrimaryEntityID:            t.PrimaryEntityID,
		Subtitle:                   t.LastMessageSummary,
		LastMessageTimestamp:       t.LastMessageTimestamp,
		Unread:                     t.Unread,
		MessageCount:               int(t.MessageCount),
		LastPrimaryEntityEndpoints: make([]*models.Endpoint, len(t.LastPrimaryEntityEndpoints)),
	}
	for i, ep := range t.LastPrimaryEntityEndpoints {
		th.LastPrimaryEntityEndpoints[i] = &models.Endpoint{
			Channel: ep.Channel.String(),
			ID:      ep.ID,
		}
	}
	return th, nil
}

func transformThreadItemToResponse(item *threading.ThreadItem, uuid, accountID string, mediaSigner *media.Signer) (*models.ThreadItem, error) {
	it := &models.ThreadItem{
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
		m2 := &models.Message{
			ThreadItemID:  item.ID,
			SummaryMarkup: m.Title,
			TextMarkup:    m.Text,
			Source: &models.Endpoint{
				Channel: m.Source.Channel.String(),
				ID:      m.Source.ID,
			},
			// TODO: EditorEntityID
			// TODO: EditedTimestamp
		}
		for _, r := range m.TextRefs {
			m2.Refs = append(m2.Refs, &models.Reference{
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
				data = &models.AudioAttachment{
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
				data = &models.ImageAttachment{
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
			m2.Attachments = append(m2.Attachments, &models.Attachment{
				Title: a.Title,
				URL:   a.URL,
				Data:  data,
			})
		}
		for _, dc := range m.Destinations {
			m2.Destinations = append(m2.Destinations, &models.Endpoint{
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

func transformSavedQueryToResponse(sq *threading.SavedQuery) (*models.SavedThreadQuery, error) {
	return &models.SavedThreadQuery{
		ID:             sq.ID,
		OrganizationID: sq.OrganizationID,
	}, nil
}

func transformEntityToResponse(e *directory.Entity) (*models.Entity, error) {
	oc, err := transformContactsToResponse(e.Contacts)
	if err != nil {
		return nil, errors.Trace(fmt.Errorf("failed to transform contacts for entity %s: %s", e.ID, err))
	}
	ent := &models.Entity{
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
		IsInternal:    e.Type == directory.EntityType_INTERNAL,
	}
	if ent.DisplayName == "" {
		// TODO: the display name will eventually be generated in the diretory service but for now this is a safety check since this must never be empty
		ent.DisplayName, err = buildDisplayName(e.Info, e.Contacts)
		if err != nil {
			golog.Errorf("Failed to generate display name for entity %s: %s", e.ID, err)
		}
		if ent.DisplayName == "" {
			ent.DisplayName = e.ID
		}
	}
	switch e.Type {
	case directory.EntityType_SYSTEM:
		ent.Avatar = &models.Image{
			URL:    "http://carefront-static.s3.amazonaws.com/icon_rx_large?system",
			Width:  72,
			Height: 72,
		}
	case directory.EntityType_ORGANIZATION:
		ent.Avatar = &models.Image{
			URL:    "http://carefront-static.s3.amazonaws.com/icon_rx_large?org",
			Width:  72,
			Height: 72,
		}
	case directory.EntityType_EXTERNAL:
		// For external entities without names we use contact info for the name. In that case we want an icon for an avatar matching the type of contact.
		// TODO: this is checking for an entity that's using contact info for the displayName. this needs
		//       to be handled better going forward but can't think of a better way for now.
		if ent.FirstName == "" || ent.LastName == "" {
			for _, c := range e.Contacts {
				switch c.ContactType {
				case directory.ContactType_PHONE:
					ent.Avatar = &models.Image{
						URL:    "https://carefront-static.s3.amazonaws.com/icon_rx_large?phone",
						Width:  72,
						Height: 72,
					}
				case directory.ContactType_EMAIL:
					ent.Avatar = &models.Image{
						URL:    "https://carefront-static.s3.amazonaws.com/icon_rx_large?email",
						Width:  72,
						Height: 72,
					}
				default:
					ent.Avatar = &models.Image{
						URL:    "https://carefront-static.s3.amazonaws.com/icon_rx_large?other",
						Width:  72,
						Height: 72,
					}
				}
				break
			}
		}
	}
	return ent, nil
}

func transformOrganizationToResponse(org *directory.Entity, provider *directory.Entity) (*models.Organization, error) {
	o := &models.Organization{
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

func transformThreadItemViewDetailsToResponse(tivds []*threading.ThreadItemViewDetails) ([]*models.ThreadItemViewDetails, error) {
	rivds := make([]*models.ThreadItemViewDetails, len(tivds))
	for i, tivd := range tivds {
		rivds[i] = &models.ThreadItemViewDetails{
			ThreadItemID:  tivd.ThreadItemID,
			ActorEntityID: tivd.EntityID,
			ViewTime:      tivd.ViewTime,
		}
	}
	return rivds, nil
}

func transformStringListSettingToResponse(config *settings.Config, value *settings.Value) *models.StringListSetting {
	return &models.StringListSetting{
		Key:         config.Key,
		Subkey:      value.Key.Subkey,
		Title:       config.Title,
		Description: config.Description,
		Value: &models.StringListSettingValue{
			Values: value.GetStringList().Values,
		},
	}
}

func transformBooleanSettingToResponse(config *settings.Config, value *settings.Value) *models.BooleanSetting {
	return &models.BooleanSetting{
		Key:         config.Key,
		Subkey:      value.Key.Subkey,
		Title:       config.Title,
		Description: config.Description,
		Value: &models.BooleanSettingValue{
			Value: value.GetBoolean().Value,
		},
	}
}

func transformMultiSelectToResponse(config *settings.Config, value *settings.Value) *models.SelectSetting {
	ss := &models.SelectSetting{
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

	ss.Options = make([]*models.SelectableItem, len(items))
	ss.Value = &models.SelectableSettingValue{
		Items: make([]*models.SelectableItemValue, len(values)),
	}

	for i, option := range items {
		ss.Options[i] = &models.SelectableItem{
			ID:            option.ID,
			Label:         option.Label,
			AllowFreeText: option.AllowFreeText,
		}
	}

	for i, v := range values {
		ss.Value.Items[i] = &models.SelectableItemValue{
			ID:   v.ID,
			Text: v.FreeTextResponse,
		}
	}

	return ss
}

func transformEntityContactToEndpoint(c *directory.Contact) (*models.Endpoint, error) {
	var channel string
	var displayValue string
	var err error
	switch c.ContactType {
	case directory.ContactType_EMAIL:
		channel = models.EndpointChannelEmail
		displayValue = c.Value
	case directory.ContactType_PHONE:
		channel = models.EndpointChannelSMS
		displayValue, err = phone.Format(c.Value, phone.Pretty)
		if err != nil {
			return nil, errors.Trace(err)
		}
	default:
		return nil, fmt.Errorf("unknown contact type %v", c.ContactType)
	}
	return &models.Endpoint{
		Channel:      channel,
		ID:           c.Value,
		DisplayValue: displayValue,
	}, nil
}
