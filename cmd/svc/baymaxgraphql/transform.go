package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/bml"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	lmedia "github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/care"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/svc/media"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
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

func transformCarePlanToResponse(cp *care.CarePlan) (*models.CarePlan, error) {
	cpr := &models.CarePlan{
		ID:                 cp.ID,
		Name:               cp.Name,
		Instructions:       make([]*models.CarePlanInstruction, len(cp.Instructions)),
		Treatments:         make([]*models.CarePlanTreatment, len(cp.Treatments)),
		CreatedTimestamp:   cp.CreatedTimestamp,
		Submitted:          cp.Submitted,
		SubmittedTimestamp: cp.SubmittedTimestamp,
		ParentID:           cp.ParentID,
		CreatorID:          cp.CreatorID,
	}
	for i, ins := range cp.Instructions {
		cpr.Instructions[i] = &models.CarePlanInstruction{
			Title: ins.Title,
			Steps: ins.Steps,
		}
	}
	for i, t := range cp.Treatments {
		cpr.Treatments[i] = &models.CarePlanTreatment{
			EPrescribe:           t.EPrescribe,
			Name:                 t.Name,
			Form:                 t.Form,
			Route:                t.Route,
			Availability:         t.Availability.String(),
			Dosage:               t.Dosage,
			DispenseType:         t.DispenseType,
			DispenseNumber:       int(t.DispenseNumber),
			Refills:              int(t.Refills),
			SubstitutionsAllowed: t.SubstitutionsAllowed,
			DaysSupply:           int(t.DaysSupply),
			Sig:                  t.Sig,
			PharmacyInstructions: t.PharmacyInstructions,
		}
	}
	return cpr, nil
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
			return nil, errors.Errorf("unsupported contact type %s", c.ContactType.String())
		}
		cs[i] = ci
	}
	return cs, nil
}

func transformThreadToResponse(ctx context.Context, ram raccess.ResourceAccessor, t *threading.Thread, viewingAccount *auth.Account) (*models.Thread, error) {
	th := &models.Thread{
		ID: t.ID,
		AllowInternalMessages:      allowInternalMessages(t, viewingAccount),
		AllowMentions:              allowMentions(t, viewingAccount),
		AllowSMSAttachments:        true,
		AllowEmailAttachment:       true,
		AllowVideoAttachment:       allowVideoAttachments(t),
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
		TypeIndicator:              threadTypeIndicator(t, viewingAccount),
		Title:                      threadTitle(ctx, ram, t, viewingAccount),
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
		th.IsTeamThread = true
		th.Type = models.ThreadTypeTeam
	case threading.ThreadType_EXTERNAL:
		th.AllowDelete = true
		th.AllowExternalDelivery = true
		th.IsPatientThread = true
		th.Type = models.ThreadTypeExternal
	case threading.ThreadType_SECURE_EXTERNAL:
		th.Type = models.ThreadTypeSecureExternal
		th.IsPatientThread = true
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
	case threading.ThreadType_LEGACY_TEAM:
		th.Type = models.ThreadTypeLegacyTeam
		th.IsTeamThread = true
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

func threadTypeIndicator(t *threading.Thread, acc *auth.Account) string {
	if acc.Type != auth.AccountType_PATIENT {
		switch t.Type {
		case threading.ThreadType_SECURE_EXTERNAL:
			return models.ThreadTypeIndicatorLock
		case threading.ThreadType_TEAM:
			return models.ThreadTypeIndicatorGroup
		}
	}
	return models.ThreadTypeIndicatorNone
}

func allowVideoAttachments(t *threading.Thread) bool {
	switch t.Type {
	case threading.ThreadType_TEAM,
		threading.ThreadType_SECURE_EXTERNAL,
		threading.ThreadType_SUPPORT,
		threading.ThreadType_LEGACY_TEAM:
		return true
	}
	return false
}

func allowMentions(t *threading.Thread, acc *auth.Account) bool {
	switch t.Type {
	case threading.ThreadType_TEAM:
		return true
	case threading.ThreadType_EXTERNAL:
		return true
	case threading.ThreadType_SECURE_EXTERNAL:
		return acc.Type == auth.AccountType_PROVIDER
	case threading.ThreadType_LEGACY_TEAM:
		return true
	case threading.ThreadType_SUPPORT:
		return t.OrganizationID == *flagSpruceOrgID
	}
	return false
}

func allowInternalMessages(t *threading.Thread, acc *auth.Account) bool {
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

func threadTitle(ctx context.Context, ram raccess.ResourceAccessor, t *threading.Thread, acc *auth.Account) string {
	if acc.Type != auth.AccountType_PATIENT {
		return t.UserTitle
	}

	org, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: t.OrganizationID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		// Log it and return the user title, don't block
		golog.Errorf("Failed to get org entity %s for thread %s to populate patient thread title", t.PrimaryEntityID, t.ID)
		return t.UserTitle
	}
	return org.Info.DisplayName
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
			esm, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
				LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
				LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
					EntityID: t.PrimaryEntityID,
				},
				Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
				RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
			})
			if err != nil {
				// Just log it. Don't block the thread
				golog.Errorf("Failed to get primary entity %s for thread %s to populate empty state markup: %s", t.PrimaryEntityID, t.ID, err)
			} else {
				return fmt.Sprintf("We've sent an invitation to %s to download the Spruce application and connect with you. You can message the patient below -- we recommend sending a personal welcome to kick things off.\n\nYou can also make internal notes about the patient’s care. These are not sent to the patient but are visible to you and your teammates.", esm.Info.DisplayName)
			}
		} else if viewingAccount.Type == auth.AccountType_PATIENT {
			esm, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
				LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
				LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
					EntityID: t.OrganizationID,
				},
				Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
				RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
			})
			if err != nil {
				// Just log it. Don't block the thread
				golog.Errorf("Failed to get organization entity %s for thread %s to populate empty state markup: %s", t.OrganizationID, t.ID, err)
			} else {
				return fmt.Sprintf("Welcome to your conversation with %s.", esm.Info.DisplayName)
			}
		}
	}
	return ""
}

func transformThreadItemToResponse(item *threading.ThreadItem, uuid, accountID, webDomain, mediaAPIDomain string) (*models.ThreadItem, error) {
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

				mediaID, err := lmedia.ParseMediaID(d.MediaID)
				if err != nil {
					golog.Errorf("Unable to parse mediaID out of url %s", d.MediaID)
				}

				a.URL = media.URL(mediaAPIDomain, mediaID)
				duration := float64(d.DurationNS) / 1e9
				data = &models.AudioAttachment{
					Mimetype:          d.Mimetype,
					URL:               a.URL,
					DurationInSeconds: duration,
				}
				// TODO
				if a.Title == "" {
					a.Title = "Audio"
				}

			case threading.Attachment_IMAGE:
				d := a.GetImage()
				if d.Mimetype == "" { // TODO
					d.Mimetype = "image/jpeg"
				}

				mediaID, err := lmedia.ParseMediaID(d.MediaID)
				if err != nil {
					golog.Errorf("Unable to parse mediaID out of url %s", d.MediaID)
				}
				a.URL = media.URL(mediaAPIDomain, mediaID)
				data = &models.ImageAttachment{
					Mimetype:     d.Mimetype,
					URL:          a.URL,
					ThumbnailURL: media.ThumbnailURL(mediaAPIDomain, mediaID, 0, 0, false),
					MediaID:      mediaID,
				}
				// TODO
				if a.Title == "" {
					a.Title = "Photo"
				}

			case threading.Attachment_VISIT:
				v := a.GetVisit()
				data = &models.BannerButtonAttachment{
					Title:   v.VisitName,
					CTAText: "View Visit",
					TapURL:  deeplink.VisitURL(webDomain, item.ThreadID, v.VisitID),
					IconURL: "https://dlzz6qy5jmbag.cloudfront.net/caremessenger/icon_visit.png",
				}
			case threading.Attachment_VIDEO:
				v := a.GetVideo()
				a.URL = media.URL(mediaAPIDomain, v.MediaID)
				data = &models.VideoAttachment{
					Mimetype:     v.Mimetype,
					URL:          a.URL,
					ThumbnailURL: media.ThumbnailURL(mediaAPIDomain, v.MediaID, 0, 0, false),
				}
			case threading.Attachment_CARE_PLAN:
				cp := a.GetCarePlan()
				a.URL = deeplink.CarePlanURL(webDomain, item.ThreadID, cp.CarePlanID)
				data = &models.BannerButtonAttachment{
					Title:   cp.CarePlanName,
					CTAText: "View Care Plan",
					TapURL:  a.URL,
					IconURL: "https://dlzz6qy5jmbag.cloudfront.net/caremessenger/icon_careplan.png",
				}

			case threading.Attachment_GENERIC_URL:
				d := a.GetGenericURL()

				// append to message
				if d.Mimetype == "application/pdf" {
					mediaID, err := lmedia.ParseMediaID(d.URL)
					if err != nil {
						golog.Errorf("Unable to parse mediaID out of url %s", d.URL)
						continue
					}

					title := a.Title
					if title == "" {
						title = "PDF Attachment"
					}

					a.URL = media.URL(mediaAPIDomain, mediaID)
					pdfAttachment := &bml.Anchor{
						HREF: a.URL,
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
				return nil, errors.Errorf("unknown attachment type %s", a.Type.String())
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
		return nil, errors.Errorf("unknown thread item type %s", item.Type.String())
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
		return nil, errors.Errorf("failed to transform contacts for entity %s: %s", e.ID, err)
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
		IsEditable:            canEditEntity(e, viewingAccount, sh),
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
		AllowEdit:             canEditEntity(e, viewingAccount, sh),
		HasPendingInvite:      entityHasPendingInvite(e),
		ImageMediaID:          e.ImageMediaID,
		HasProfile:            e.HasProfile,
	}

	if viewingAccount.Type == auth.AccountType_PROVIDER {
		ent.CallableEndpoints, err = callableEndpointsForEntity(e)
		if err != nil {
			return nil, errors.Trace(err)
		}
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

func entityHasPendingInvite(ent *directory.Entity) bool {
	// TODO: We should actually look this up. But for now we can do a poor man's version.
	// If an entity is a Patient and has not created an account then in the current system
	// they MUST have a pending invite.
	return ent.Type == directory.EntityType_PATIENT && ent.AccountID == ""
}

func canEditEntity(e *directory.Entity, viewingAccount *auth.Account, sh *device.SpruceHeaders) bool {
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
		if sh.Platform == device.IOS {
			return (sh.AppVersion != nil && !sh.AppVersion.Equals(&encoding.Version{Major: 1}))
		}
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

func transformVisitCategoryToResponse(vc *layout.VisitCategory) *models.VisitCategory {
	return &models.VisitCategory{
		ID:   vc.ID,
		Name: vc.Name,
	}
}

func transformVisitLayoutToResponse(vl *layout.VisitLayout) *models.VisitLayout {
	return &models.VisitLayout{
		ID:   vl.ID,
		Name: vl.Name,
	}
}

func transformVisitLayoutVersionToResponse(version *layout.VisitLayoutVersion, store layout.Storage) (*models.VisitLayoutVersion, error) {

	par := conc.NewParallel()

	var samlLayout []byte
	var layoutPreview []byte
	par.Go(func() error {

		samlIntake, err := store.GetSAML(version.SAMLLocation)
		if err != nil {
			return errors.Trace(err)
		}

		samlLayout, err = json.Marshal(samlIntake)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	})

	par.Go(func() error {

		intake, err := store.GetIntake(version.IntakeLayoutLocation)
		if err != nil {
			return errors.Trace(err)
		}

		review, err := store.GetReview(version.ReviewLayoutLocation)
		if err != nil {
			return errors.Trace(err)
		}

		intakePreview, err := care.GenerateVisitLayoutPreview(intake, review)
		if err != nil {
			return errors.Trace(err)
		}

		layoutPreview, err = json.Marshal(intakePreview)
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	})

	if err := par.Wait(); err != nil {
		return nil, errors.Trace(err)
	}

	return &models.VisitLayoutVersion{
		ID:            version.ID,
		SAMLLayout:    string(samlLayout),
		LayoutPreview: string(layoutPreview),
	}, nil
}

type byVisitLayoutName []*layout.VisitLayout

func (c byVisitLayoutName) Len() int      { return len(c) }
func (c byVisitLayoutName) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c byVisitLayoutName) Less(i, j int) bool {
	return strings.Compare(strings.ToLower(c[i].Name), strings.ToLower(c[j].Name)) < 0
}

type byVisitCategoryName []*layout.VisitCategory

func (c byVisitCategoryName) Len() int      { return len(c) }
func (c byVisitCategoryName) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c byVisitCategoryName) Less(i, j int) bool {
	return strings.Compare(strings.ToLower(c[i].Name), strings.ToLower(c[j].Name)) < 0
}

func transformVisitToResponse(ctx context.Context, ram raccess.ResourceAccessor, orgEntity *directory.Entity, visit *care.Visit, layoutVersion *layout.VisitLayoutVersion, layoutStore layout.Storage) (*models.Visit, error) {

	acc := gqlctx.Account(ctx)
	if acc == nil {
		return nil, errors.Errorf("expected acccount to not be nil but it was")
	}

	intake, err := layoutStore.GetIntake(layoutVersion.IntakeLayoutLocation)
	if err != nil {
		return nil, errors.Trace(err)
	}

	answersForVisitRes, err := ram.GetAnswersForVisit(ctx, &care.GetAnswersForVisitRequest{
		VisitID:              visit.ID,
		SerializedForPatient: acc.Type == auth.AccountType_PATIENT,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	var containerData []byte
	var layoutContainerType string
	switch acc.Type {
	case auth.AccountType_PATIENT:
		layoutContainerType = layoutContainerTypeIntake
		containerData, err = care.PopulateVisitIntake(intake, &care.VisitData{
			PatientAnswersJSON: []byte(answersForVisitRes.PatientAnswersJSON),
			Visit:              visit,
			OrgEntity:          orgEntity,
			Preferences: map[string]interface{}{
				"optional_triage": visit.Preferences.OptionalTriage,
			},
		})
		if err != nil {
			return nil, errors.Trace(err)
		}

	case auth.AccountType_PROVIDER:
		layoutContainerType = layoutContainerTypeReview
		review, err := layoutStore.GetReview(layoutVersion.ReviewLayoutLocation)
		if err != nil {
			return nil, errors.Trace(err)
		}

		containerData, err = care.PopulateVisitReview(intake, review, answersForVisitRes.Answers, visit)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	return &models.Visit{
		ID:                  visit.ID,
		EntityID:            visit.EntityID,
		Name:                visit.Name,
		CanReview:           visit.Submitted || visit.Triaged,
		CanPatientModify:    acc.Type == auth.AccountType_PATIENT && !visit.Submitted && !visit.Triaged,
		Submitted:           visit.Submitted,
		Triaged:             visit.Triaged,
		LayoutContainer:     string(containerData),
		LayoutContainerType: layoutContainerType,
	}, nil
}

func transformProfileToResponse(ctx context.Context, ram raccess.ResourceAccessor, p *directory.Profile) *models.Profile {
	var title string
	var allowEdit bool
	parallel := conc.NewParallel()
	parallel.Go(func() error {
		title = profileTitle(ctx, ram, p)
		return nil
	})
	parallel.Go(func() error {
		allowEdit = raccess.ProfileAllowEdit(ctx, ram, p.EntityID)
		return nil
	})
	// don't concern ourselves with errors here, just default
	parallel.Wait()
	return &models.Profile{
		ID:                    p.ID,
		EntityID:              p.EntityID,
		Title:                 title,
		Sections:              transformProfileSectionsResponses(p.Sections),
		AllowEdit:             allowEdit,
		LastModifiedTimestamp: p.LastModifiedTimestamp,
	}
}

func transformProfileSectionsResponses(pss []*directory.ProfileSection) []*models.ProfileSection {
	rPSs := make([]*models.ProfileSection, len(pss))
	for i, ps := range pss {
		rPSs[i] = &models.ProfileSection{
			Title: ps.Title,
			Body:  ps.Body,
		}
	}
	return rPSs
}

func profileTitle(ctx context.Context, ram raccess.ResourceAccessor, p *directory.Profile) string {
	// We need the owning entity to get the display title
	ent, err := raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: p.EntityID,
		},
	})
	// Any error means empty title
	if err != nil {
		golog.Errorf("Encountered error while generating title for Profile %s: %s", p.ID, err)
		return ""
	}
	return ent.Info.DisplayName
}
