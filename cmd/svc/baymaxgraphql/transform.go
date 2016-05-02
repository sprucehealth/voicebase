package main

import (
	"fmt"
	"strings"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/media"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

func transformAccountToResponse(a *auth.Account) models.Account {
	if a == nil {
		return nil
	}
	switch a.Type {
	case auth.AccountType_PROVIDER:
		return &models.ProviderAccount{
			ID: a.ID,
		}
	case auth.AccountType_PATIENT:
		return &models.PatientAccount{
			ID: a.ID,
		}
	}
	golog.Errorf("Unable to transform account of type %s to repsonse", a.Type.String())
	return nil
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

func transformThreadToResponse(ctx context.Context, ram raccess.ResourceAccessor, t *threading.Thread, viewingAccount *auth.Account) (*models.Thread, error) {
	th := &models.Thread{
		ID: t.ID,
		AllowInternalMessages:      isPostInternalMessageAllowed(t, viewingAccount),
		OrganizationID:             t.OrganizationID,
		PrimaryEntityID:            t.PrimaryEntityID,
		Subtitle:                   t.LastMessageSummary,
		LastMessageTimestamp:       t.LastMessageTimestamp,
		Unread:                     t.Unread,
		UnreadReference:            t.UnreadReference,
		MessageCount:               int(t.MessageCount),
		LastPrimaryEntityEndpoints: make([]*models.Endpoint, len(t.LastPrimaryEntityEndpoints)),
		EmptyStateTextMarkup:       threadEmptyStateTextMarkup(ctx, ram, t, viewingAccount),
		Type:                       t.Type.String(),
		TypeIndicator:              threadTypeIndicator(t),
		Title:                      t.UserTitle,
	}
	if th.Title == "" {
		th.Title = t.SystemTitle
	}

	switch t.Type {
	case threading.ThreadType_TEAM:
		th.AllowAddMembers = true
		th.AllowLeave = true
		th.AllowRemoveMembers = true
		th.AllowUpdateTitle = true
		th.AllowMentions = true
		th.Type = models.ThreadTypeTeam
	case threading.ThreadType_EXTERNAL:
		th.AllowDelete = true
		th.AllowExternalDelivery = true
		th.AllowMentions = true
		th.Type = models.ThreadTypeExternal
	case threading.ThreadType_SECURE_EXTERNAL:
		th.Type = models.ThreadTypeSecureExternal
	case threading.ThreadType_SETUP:
		if th.Title == "" {
			th.Title = onboardingThreadTitle
		}
		th.Type = models.ThreadTypeSetup
	case threading.ThreadType_SUPPORT:
		if th.Title == "" {
			th.Title = supportThreadTitle
		}
		th.Type = models.ThreadTypeSupport

		// only allow @ mentions if we are working in the spruce support thread.
		th.AllowMentions = th.OrganizationID == *flagSpruceOrgID
	case threading.ThreadType_LEGACY_TEAM:
		th.Type = models.ThreadTypeLegacyTeam
		th.AllowMentions = true
	case threading.ThreadType_UNKNOWN: // TODO: remove this once old threads are migrated
		th.Type = models.ThreadTypeUnknown
	default:
		return nil, fmt.Errorf("Unknown thread type %s", t.Type)
	}
	for i, ep := range t.LastPrimaryEntityEndpoints {
		th.LastPrimaryEntityEndpoints[i] = &models.Endpoint{
			Channel: ep.Channel.String(),
			ID:      ep.ID,
		}
	}
	return th, nil
}

func threadTypeIndicator(t *threading.Thread) string {
	switch t.Type {
	case threading.ThreadType_SECURE_EXTERNAL:
		return models.ThreadTypeIndicatorLock
	case threading.ThreadType_TEAM:
		return models.ThreadTypeIndicatorGroup
	}
	return models.ThreadTypeIndicatorNone
}

func isPostInternalMessageAllowed(t *threading.Thread, acc *auth.Account) bool {
	switch t.Type {
	case threading.ThreadType_EXTERNAL:
		return true
	case threading.ThreadType_SECURE_EXTERNAL:
		return acc.Type == auth.AccountType_PROVIDER
	case threading.ThreadType_SETUP:
		return true
	case threading.ThreadType_SUPPORT:
		return t.OrganizationID == *flagSpruceOrgID
	}
	return false
}

func threadEmptyStateTextMarkup(ctx context.Context, ram raccess.ResourceAccessor, t *threading.Thread, viewingAccount *auth.Account) string {
	if t.MessageCount != 0 {
		return ""
	}
	switch t.Type {
	case threading.ThreadType_TEAM:
		return "This is the beginning of your team conversation.\nSend a message to get things started."
	case threading.ThreadType_SECURE_EXTERNAL:
		if viewingAccount.Type == auth.AccountType_PROVIDER {
			esm, err := ram.Entity(ctx, t.PrimaryEntityID, nil, 0)
			if err != nil {
				// Just log it. Don't block the thread
				golog.Errorf("Failed to get primary entity %s for thread %s to populate empty state markup", t.PrimaryEntityID, t.ID)
			} else {
				return fmt.Sprintf("We've sent an invitation to %s to create a Spruce account and join the conversation.\n\nWe recommend sending a personal welcome message to kick things off.", esm.Info.DisplayName)
			}
		}
	}
	return ""
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
			// TODO: EditorEntityID
			// TODO: EditedTimestamp
		}
		if m.Source != nil {
			m2.Source = &models.Endpoint{
				Channel: m.Source.Channel.String(),
				ID:      m.Source.ID,
			}
		} else {
			// TODO: for now setting source to APP if not included since clients might assume it's always included
			m2.Source = &models.Endpoint{
				Channel: threading.Endpoint_APP.String(),
				ID:      item.ActorEntityID,
			}
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
				duration := float64(d.DurationNS) / 1e9
				data = &models.AudioAttachment{
					Mimetype:          d.Mimetype,
					URL:               signedURL,
					DurationInSeconds: duration,
				}
				// TODO
				if a.Title == "" {
					a.Title = "Audio"
				}
				a.URL = signedURL
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
				a.URL = signedURL
			case threading.Attachment_GENERIC_URL:
				d := a.GetGenericURL()

				// append to message
				if d.Mimetype == "application/pdf" {
					mediaID, err := media.ParseMediaID(d.URL)
					if err != nil {
						golog.Errorf("Unable to parse mediaID out of url %s", d.URL)
						continue
					}

					signedURL, err := mediaSigner.SignedURL(mediaID, d.Mimetype, accountID, 0, 0, false)
					if err != nil {
						golog.Errorf("Unable to generate signed url for media %s: %s", mediaID, err.Error())
						continue
					}

					title := a.Title
					if title == "" {
						title = "PDF Attachment"
					}

					pdfAttachment := &bml.Anchor{
						HREF: signedURL,
						Text: title,
					}

					textMarkup, err := bml.Parse(m2.TextMarkup)
					if err != nil {
						// should not error because coming straight from the database and expected to be clean
						return nil, errors.Trace(err)
					}
					textMarkup = append(textMarkup, "\n\n", pdfAttachment, "\n")

					m2.TextMarkup, err = textMarkup.Format()
					if err != nil {
						// shouldn't fail
						return nil, errors.Trace(err)
					}

				} else {
					golog.Warningf("Dropping attachment because mimetype %s for thread item %s is not supported", d.Mimetype, item.ID)
				}
				continue
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

func transformEntityToResponse(staticURLPrefix string, e *directory.Entity, sh *device.SpruceHeaders, viewingAccount *auth.Account) (*models.Entity, error) {
	oc, err := transformContactsToResponse(e.Contacts)
	if err != nil {
		return nil, errors.Trace(fmt.Errorf("failed to transform contacts for entity %s: %s", e.ID, err))
	}

	isEditable := false
	if e.Type != directory.EntityType_SYSTEM && sh != nil {
		if sh.Platform == device.IOS {
			isEditable = sh.AppVersion != nil && !sh.AppVersion.Equals(&encoding.Version{Major: 1})
		} else {
			isEditable = true
		}
	}

	var dob *models.DOB
	if e.Info.DOB != nil {
		dob = &models.DOB{
			Month: int(e.Info.DOB.Month),
			Day:   int(e.Info.DOB.Day),
			Year:  int(e.Info.DOB.Year),
		}
	}

	ent := &models.Entity{
		ID:                    e.ID,
		IsEditable:            isEditable,
		Contacts:              oc,
		FirstName:             e.Info.FirstName,
		MiddleInitial:         e.Info.MiddleInitial,
		LastName:              e.Info.LastName,
		GroupName:             e.Info.GroupName,
		DisplayName:           e.Info.DisplayName,
		ShortTitle:            e.Info.ShortTitle,
		LongTitle:             e.Info.LongTitle,
		Gender:                e.Info.Gender.String(),
		DOB:                   dob,
		Note:                  e.Info.Note,
		IsInternal:            e.Type == directory.EntityType_INTERNAL,
		LastModifiedTimestamp: e.LastModifiedTimestamp,
		HasAccount:            e.AccountID != "",
		AllowEdit:             canEditEntity(e, viewingAccount),
	}

	switch e.Type {
	case directory.EntityType_ORGANIZATION:
		ent.Avatar = &models.Image{
			URL:    staticURLPrefix + "img/avatar/icon_profile_spruceassist@3x.png",
			Width:  108,
			Height: 108,
		}
	case directory.EntityType_SYSTEM:
		// TODO: it is brittle to use the name for checking the difference, but right now there's no other way to know
		if ent.DisplayName == supportThreadTitle {
			ent.Avatar = &models.Image{
				URL:    staticURLPrefix + "img/avatar/icon_profile_teamspruce@3x.png",
				Width:  108,
				Height: 108,
			}
		} else {
			ent.Avatar = &models.Image{
				URL:    staticURLPrefix + "img/avatar/icon_profile_spruceassist@3x.png",
				Width:  108,
				Height: 108,
			}
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
						URL:    staticURLPrefix + "img/avatar/icon_profile_phone@3x.png",
						Width:  108,
						Height: 108,
					}
				case directory.ContactType_EMAIL:
					ent.Avatar = &models.Image{
						URL:    staticURLPrefix + "img/avatar/icon_profile_email@3x.png",
						Width:  108,
						Height: 108,
					}
				default:
					ent.Avatar = &models.Image{
						URL:    staticURLPrefix + "img/avatar/icon_profile_user@3x.png",
						Width:  108,
						Height: 108,
					}
				}
				break
			}
		}
	}
	return ent, nil
}

func canEditEntity(e *directory.Entity, viewingAccount *auth.Account) bool {
	// An unauthenticated use can never can never edit
	if viewingAccount == nil || e == nil {
		return false
	}

	// If the viewer owns the entity then they can always edit it
	if e.AccountID == viewingAccount.ID {
		return true
	}

	// If the viewer is a provider and the entity is an external entity then they can edit
	if viewingAccount.Type == auth.AccountType_PROVIDER &&
		e.Type == directory.EntityType_EXTERNAL {
		return true
	}

	return false
}

func transformOrganizationToResponse(staticURLPrefix string, org *directory.Entity, provider *directory.Entity, sh *device.SpruceHeaders, viewingAccount *auth.Account) (*models.Organization, error) {
	o := &models.Organization{
		ID:   org.ID,
		Name: org.Info.DisplayName,
	}

	oc, err := transformContactsToResponse(org.Contacts)
	if err != nil {
		return nil, fmt.Errorf("failed to transform entity contacts: %+v", err)
	}

	o.Contacts = oc

	e, err := transformEntityToResponse(staticURLPrefix, provider, sh, viewingAccount)
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
		Key:                     config.Key,
		Subkey:                  value.Key.Subkey,
		Title:                   config.Title,
		Description:             config.Description,
		AllowsMultipleSelection: config.Type == settings.ConfigType_MULTI_SELECT,
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
