package twilio

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func determineExternalEntityName(ctx context.Context, source phone.Number, organizationID string, eh *eventsHandler) (string, error) {
	// determine the external entity if possible so that we can announce their name
	res, err := eh.directory.LookupEntitiesByContact(
		ctx,
		&directory.LookupEntitiesByContactRequest{
			ContactValue: source.String(),
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERSHIPS,
				},
			},
			Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes: []directory.EntityType{directory.EntityType_EXTERNAL},
		})
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return "", nil
		}
		return "", errors.Trace(err)
	}
	for _, e := range res.Entities {

		// find the entity that has a membership to the organization
		for _, m := range e.Memberships {
			if m.ID == organizationID {
				return e.Info.DisplayName, nil
			}
		}
	}
	return "", nil
}
