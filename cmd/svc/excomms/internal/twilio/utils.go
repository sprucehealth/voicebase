package twilio

import (
	"context"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func determinePatientName(ctx context.Context, source phone.Number, organizationID string, eh *eventsHandler) (string, error) {
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
			Statuses:       []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			RootTypes:      []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_PATIENT},
			MemberOfEntity: organizationID,
		})
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			return "", nil
		}
		return "", errors.Trace(err)
	} else if len(res.Entities) == 0 {
		return "", nil
	}

	return res.Entities[0].Info.DisplayName, nil
}
