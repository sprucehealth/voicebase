package worker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/operational/internal/dal"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/operational"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// BlockAccountWorker is responsible for doing the work associated
// with the act of blocking a particular account for misuse of the system.
type BlockAccountWorker struct {
	auth        auth.AuthClient
	directory   directory.DirectoryClient
	excomms     excomms.ExCommsClient
	threading   threading.ThreadsClient
	sqs         sqsiface.SQSAPI
	dal         dal.DAL
	worker      *awsutil.SQSWorker
	spruceOrgID string
}

type snsMessage struct {
	Message []byte
}

func NewBlockAccountWorker(
	auth auth.AuthClient,
	directory directory.DirectoryClient,
	excomms excomms.ExCommsClient,
	threading threading.ThreadsClient,
	sqs sqsiface.SQSAPI,
	dal dal.DAL,
	blockAccountSQSURL,
	spruceOrgID string) *BlockAccountWorker {
	w := &BlockAccountWorker{
		auth:        auth,
		directory:   directory,
		excomms:     excomms,
		threading:   threading,
		sqs:         sqs,
		dal:         dal,
		spruceOrgID: spruceOrgID,
	}
	w.worker = awsutil.NewSQSWorker(sqs, blockAccountSQSURL, w.processSNSEvent)
	return w
}

func (w *BlockAccountWorker) Start() {
	w.worker.Start()
}

func (w *BlockAccountWorker) Stop(wait time.Duration) {
	w.worker.Stop(wait)
}

func (w *BlockAccountWorker) processSNSEvent(ctx context.Context, msg string) error {
	var snsMsg snsMessage
	if err := json.Unmarshal([]byte(msg), &snsMsg); err != nil {
		golog.Errorf("Failed to unmarshal sns message: %s", err.Error())
		return nil
	}

	var bar operational.BlockAccountRequest
	if err := bar.Unmarshal(snsMsg.Message); err != nil {
		golog.Errorf("Failed to unmarshal block account request: %s", err.Error())
		return nil
	}

	return errors.Trace(w.processEvent(ctx, &bar))
}

func (w *BlockAccountWorker) processEvent(ctx context.Context, bar *operational.BlockAccountRequest) error {
	// block account via the auth service
	res, err := w.auth.BlockAccount(ctx, &auth.BlockAccountRequest{
		AccountID: bar.AccountID,
	})
	if err != nil {
		if grpc.Code(err) == codes.NotFound {
			// nothing to do if the account to be blocked is not found
			return nil
		}
		return errors.Trace(err)
	}
	accountID := res.Account.ID

	// lookup entity via account id
	entityLookupRes, err := w.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: accountID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
	})
	if err != nil {
		return errors.Trace(err)
	} else if len(entityLookupRes.Entities) != 1 {
		return errors.Trace(fmt.Errorf("Expected 1 entity for accountID %s but got %d", accountID, len(entityLookupRes.Entities)))
	}

	// for all the organization that the user belongs to,
	// if the user is the only member of the organization, clean up
	// the org level resources as well.
	for _, m := range entityLookupRes.Entities[0].Memberships {
		if m.Type != directory.EntityType_ORGANIZATION {
			continue
		}
		organizationID := m.ID

		// if the org only has one member, then deprovision phone number and delete spruce support thread from spruce support org
		orgLookupRes, err := w.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: organizationID,
			},
			RequestedInformation: &directory.RequestedInformation{
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
					directory.EntityInformation_MEMBERS,
				},
			},
		})
		if err != nil {
			return errors.Trace(err)
		} else if len(orgLookupRes.Entities) != 1 {
			return errors.Trace(fmt.Errorf("Expected 1 org to be returned for %s but got %d", organizationID, len(orgLookupRes.Entities)))
		}

		if determineInternalMemberCount(orgLookupRes.Entities[0]) > 1 {
			// don't delete the spruce thread or deprovision the number
			// if there is more than one member in the organization
			return nil
		}

		// TODO: Delete the spruce support thread in the spruce org.
	}

	// record the fact that the account was blocked
	if err := w.dal.MarkAccountAsBlocked(bar.AccountID); err != nil {
		golog.Errorf("Unable to mark account as blocked :%s", err.Error())
	}

	return nil
}

func determineInternalMemberCount(entity *directory.Entity) int {
	var numInternalMembers int

	for _, member := range entity.Members {
		if member.Type == directory.EntityType_INTERNAL {
			numInternalMembers++
		}
	}
	return numInternalMembers
}
