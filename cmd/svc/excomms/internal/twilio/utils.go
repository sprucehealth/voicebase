package twilio

import (
	"fmt"

	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func getForwardingListForProvisionedPhoneNumber(ctx context.Context, phoneNumber, organizationID string, eh *eventsHandler) ([]string, error) {

	settingsRes, err := eh.settings.GetValues(ctx, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyForwardingList,
				Subkey: phoneNumber,
			},
		},
		NodeID: organizationID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(settingsRes.Values) != 1 {
		return nil, errors.Trace(fmt.Errorf("Expected single value for forwarding list of provisioned phone number %s but got back %d", phoneNumber, len(settingsRes.Values)))
	} else if settingsRes.Values[0].GetStringList() == nil {
		return nil, errors.Trace(fmt.Errorf("Expected string list value but got %T", settingsRes.Values[0]))
	}

	forwardingListMap := make(map[string]bool, len(settingsRes.Values[0].GetStringList().Values))
	forwardingList := make([]string, 0, len(settingsRes.Values[0].GetStringList().Values))
	for _, s := range settingsRes.Values[0].GetStringList().Values {
		if forwardingListMap[s] {
			continue
		}
		forwardingListMap[s] = true
		forwardingList = append(forwardingList, s)
	}

	return forwardingList, nil
}

func determineExternalEntityName(ctx context.Context, source phone.Number, organizationID string, eh *eventsHandler) (string, error) {
	// determine the external entity if possible so that we can announce their name
	res, err := eh.directory.LookupEntitiesByContact(
		ctx,
		&directory.LookupEntitiesByContactRequest{
			ContactValue: source.String(),
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
					directory.EntityInformation_MEMBERSHIPS,
				},
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		})
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return "", nil
		}
		return "", errors.Trace(err)
	}
	for _, e := range res.Entities {

		// only deal with external parties
		if e.Type != directory.EntityType_EXTERNAL {
			continue
		}

		// find the entity that has a membership to the organization
		for _, m := range e.Memberships {
			if m.ID == organizationID {
				// only use the display name if the first and last name
				// exist. We use this fact as an indicator that the display name
				// is probably the name of the patient (versus phone number or email address).
				if e.Info.FirstName != "" && e.Info.LastName != "" {
					return e.Info.DisplayName, nil
				}
			}
		}
	}
	return "", nil
}

func determineEntityWithProvisionedEndpoint(eh *eventsHandler, endpoint string, depth int64) (*directory.Entity, error) {
	res, err := eh.directory.LookupEntitiesByContact(
		context.Background(),
		&directory.LookupEntitiesByContactRequest{
			ContactValue: endpoint,
			RequestedInformation: &directory.RequestedInformation{
				Depth: depth,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERS,
					directory.EntityInformation_MEMBERSHIPS,
					directory.EntityInformation_CONTACTS,
				},
			},
			Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		})
	if err != nil {
		return nil, errors.Trace(err)
	}

	// only one entity should exist with the provisioned value
	var entity *directory.Entity
	for _, e := range res.Entities {
		for _, c := range e.Contacts {
			if c.Provisioned && c.Value == endpoint {
				if entity != nil {
					return nil, errors.Trace(fmt.Errorf("More than 1 entity found with provisioned endpoint %s", endpoint))
				}

				entity = e
			}
		}
	}

	if entity == nil {
		return nil, errors.Trace(fmt.Errorf("No entity found for provisioned endpoint %s", endpoint))
	}

	return entity, nil
}
