package worker

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
)

type Service interface {
	Start()
	Shutdown() error
}

type syncEvent struct {
	dl               dal.DAL
	directory        directory.DirectoryClient
	threading        threading.ThreadsClient
	syncEventsWorker worker.Worker
	webDomain        string
}

// NewSyncEvent returns a worker that is responsible for processing messages to
// undertake sync events for a particular organization paired with a particular source (eg hint, elation, csv, drchrono, etc)
func NewSyncEvent(
	dl dal.DAL,
	directory directory.DirectoryClient,
	threading threading.ThreadsClient,
	sqsAPI sqsiface.SQSAPI,
	syncEventsSQSURL, webDomain string,
) Service {
	s := &syncEvent{
		dl:        dl,
		directory: directory,
		threading: threading,
		webDomain: webDomain,
	}
	s.syncEventsWorker = awsutil.NewSQSWorker(sqsAPI, syncEventsSQSURL, s.processSyncEvent)
	return s
}

func (s *syncEvent) Start() {
	s.syncEventsWorker.Start()
}

func (s *syncEvent) Shutdown() error {
	s.syncEventsWorker.Stop(time.Second * 30)
	return nil
}

// processSyncEvent is the core function of the worker to sync a particular event
// from an EMR to the org's inbox.
func (s *syncEvent) processSyncEvent(ctx context.Context, data string) error {
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return errors.Trace(err)
	}

	var event sync.Event
	if err := event.Unmarshal([]byte(decodedData)); err != nil {
		return errors.Trace(err)
	}

	cfg, err := s.dl.SyncConfigForOrg(event.OrganizationEntityID, event.Source.String())
	if err != nil {
		return errors.Errorf("Unable to look up sync config for org %s : %s", event.OrganizationEntityID, err)
	}

	switch event.Type {
	case sync.EVENT_TYPE_PATIENT_ADD:
		return s.processPatientAddEvent(ctx, cfg, &event)
	}

	return errors.Errorf("Unknown event type %s for org %s", event.Type.String(), event.OrganizationEntityID)
}

func (s *syncEvent) processPatientAddEvent(ctx context.Context, cfg *sync.Config, event *sync.Event) error {
	addEvent := event.GetPatientAddEvent()
	if addEvent == nil {
		return errors.Errorf("Expected add event for %s and org %s but got none", event.Type.String(), event.OrganizationEntityID)
	}

	// go through the list of patients and create the appropriate threads for each
	for _, patient := range addEvent.GetPatients() {

		sanitizePatient(patient)

		switch cfg.ThreadCreationType {
		case sync.THREAD_CREATION_TYPE_STANDARD:
			if err := s.createThread(ctx, patient, event.Source, event.OrganizationEntityID, threading.THREAD_TYPE_EXTERNAL); err != nil {
				return errors.Trace(err)
			}
		case sync.THREAD_CREATION_TYPE_SECURE:
			if err := s.createThread(ctx, patient, event.Source, event.OrganizationEntityID, threading.THREAD_TYPE_SECURE_EXTERNAL); err != nil {
				return errors.Trace(err)
			}
		default:
			return errors.Errorf("unknown thread creation type for org %s", event.OrganizationEntityID)
		}
	}

	return nil
}

func (s *syncEvent) createThread(ctx context.Context, patient *sync.Patient, source sync.Source, orgID string, threadType threading.ThreadType) error {

	// check if the patient already exists as an entity based on the patient ID
	if patient.ID == "" {
		return errors.Errorf("cannot create patient with no external ID for org %s", orgID)
	}

	entityType := directory.EntityType_EXTERNAL
	if threadType == threading.THREAD_TYPE_SECURE_EXTERNAL {
		entityType = directory.EntityType_PATIENT
	}

	res, err := s.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: sync.ExternalIDFromSource(patient.ID, source),
		},
		MemberOfEntity: orgID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{entityType},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil && grpc.Code(err) != codes.NotFound {
		return errors.Errorf("unable to lookup patient %s for org %s: %s", patient.ID, orgID, err)
	}

	var externalEntity *directory.Entity
	if res != nil && len(res.Entities) > 0 {
		externalEntity = res.Entities[0]
	} else {
		// create entity
		res, err := s.directory.CreateEntity(ctx, &directory.CreateEntityRequest{
			Type:                      entityType,
			ExternalID:                sync.ExternalIDFromSource(patient.ID, source),
			InitialMembershipEntityID: orgID,
			RequestedInformation: &directory.RequestedInformation{
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
					directory.EntityInformation_EXTERNAL_IDS,
					directory.EntityInformation_MEMBERSHIPS,
				},
			},
			Contacts: sync.ContactsFromPatient(patient),
			EntityInfo: &directory.EntityInfo{
				FirstName: patient.FirstName,
				LastName:  patient.LastName,
			},
		})
		if err != nil {
			return errors.Errorf("Unable to create entity for %s in org %s: %s", patient.ID, orgID, err)
		}
		externalEntity = res.Entity
	}

	// check if the thread for the patient already exists
	var thread *threading.Thread
	threadRes, err := s.threading.ThreadsForMember(ctx, &threading.ThreadsForMemberRequest{
		PrimaryOnly: true,
		EntityID:    externalEntity.ID,
	})
	if err != nil && grpc.Code(err) != codes.NotFound {
		return errors.Errorf("Unable to lookup threads for member %s : %s", externalEntity.ID, err)
	} else if threadRes == nil || len(threadRes.Threads) == 0 {
		// create thread
		createThreadRes, err := s.threading.CreateEmptyThread(ctx, &threading.CreateEmptyThreadRequest{
			OrganizationID:  orgID,
			PrimaryEntityID: externalEntity.ID,
			MemberEntityIDs: []string{orgID},
			Type:            threadType,
			Summary:         externalEntity.Info.DisplayName,
			SystemTitle:     externalEntity.Info.DisplayName,
			Origin:          threading.THREAD_ORIGIN_SYNC,
		})
		if err != nil {
			return errors.Errorf("Unable to create thread for %s : %s", externalEntity.ID, err)
		}
		thread = createThreadRes.Thread
	} else {
		thread = threadRes.Threads[0]
	}

	if patient.ExternalURL != "" {
		if _, err := s.directory.CreateExternalLink(ctx, &directory.CreateExternalLinkRequest{
			EntityID: externalEntity.ID,
			Name:     nameForExternalURL(source),
			URL:      patient.ExternalURL,
		}); err != nil {
			return errors.Errorf("Unable to create external link for entity %s: %s", externalEntity.ID, err)
		}
	}

	// TODO: link patient back to source (probably by posting another notification that creation is complete)
	fmt.Println(deeplink.ThreadURLShareable(s.webDomain, orgID, thread.ID))

	return nil
}
