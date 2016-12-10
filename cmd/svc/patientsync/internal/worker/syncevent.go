package worker

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/caremessenger/deeplink"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/worker"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type Service interface {
	Start()
	Shutdown() error
}

type syncEvent struct {
	dl               dal.DAL
	directory        directory.DirectoryClient
	threading        threading.ThreadsClient
	settings         settings.SettingsClient
	invite           invite.InviteClient
	syncEventsWorker worker.Worker
	webDomain        string
}

// NewSyncEvent returns a worker that is responsible for processing messages to
// undertake sync events for a particular organization paired with a particular source (eg hint, elation, csv, drchrono, etc)
func NewSyncEvent(
	dl dal.DAL,
	directory directory.DirectoryClient,
	threading threading.ThreadsClient,
	settings settings.SettingsClient,
	invite invite.InviteClient,
	sqsAPI sqsiface.SQSAPI,
	syncEventsSQSURL, webDomain string,
) Service {
	s := &syncEvent{
		dl:        dl,
		directory: directory,
		threading: threading,
		settings:  settings,
		webDomain: webDomain,
		invite:    invite,
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

	switch event.Event.(type) {
	case *sync.Event_PatientAddEvent:
		return s.processPatientAddEvent(ctx, cfg, &event)
	case *sync.Event_PatientUpdateEvent:
		return s.processPatientUpdatedEvent(ctx, cfg, &event)
	}

	return errors.Errorf("Unknown event type %s for org %s", event.Type.String(), event.OrganizationEntityID)
}

func (s *syncEvent) processPatientUpdatedEvent(ctx context.Context, cfg *sync.Config, event *sync.Event) error {
	updatedEvent := event.GetPatientUpdateEvent()

	for _, patient := range updatedEvent.GetPatients() {
		sanitizePatient(patient)

		// check if patient already exists, ignore update if patient deleted
		res, err := s.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
			Key: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: sync.ExternalIDFromSource(patient.ID, event.Source),
			},
			MemberOfEntity: cfg.OrganizationEntityID,
			RequestedInformation: &directory.RequestedInformation{
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
					directory.EntityInformation_EXTERNAL_IDS,
					directory.EntityInformation_MEMBERSHIPS,
				},
			},
			RootTypes:  []directory.EntityType{directory.EntityType_PATIENT, directory.EntityType_EXTERNAL},
			ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
		})
		if err != nil && grpc.Code(err) != codes.NotFound {
			return errors.Errorf("unable to lookup patient %s for org %s: %s", patient.ID, cfg.OrganizationEntityID, err)
		}

		var patientEntity *directory.Entity
		var entityDeleted bool
		if res != nil {
			for _, entity := range res.Entities {
				if entity.Status == directory.EntityStatus_DELETED {
					entityDeleted = true
					break
				} else if entity.Status == directory.EntityStatus_ACTIVE {
					patientEntity = entity
					break
				}
			}
		}

		if entityDeleted {
			// if the entity has been deleted, ignore the sync
			continue
		}

		// if patient does not exist, create it.
		if patientEntity == nil {
			if err := s.createPatientAndThread(ctx, patient, cfg, event); err != nil {
				return errors.Trace(err)
			}
			continue
		}

		// if patient does exist, check if object differs for the properties that matter
		if !sync.Differs(patient, patientEntity) {
			continue
		}

		// ignore update if the patient was locally updated more recently
		// than the incoming update
		if patientEntity.LastModifiedTimestamp > patient.LastModifiedTime {
			continue
		}

		patientEntity.Info.FirstName = patient.FirstName
		patientEntity.Info.LastName = patient.LastName
		patientEntity.Info.DOB = sync.TransformDOB(patient.DOB)
		patientEntity.Info.Gender = sync.TransformGender(patient.Gender)

		// if it does, update it.
		updateRes, err := s.directory.UpdateEntity(ctx, &directory.UpdateEntityRequest{
			EntityID:         patientEntity.ID,
			Contacts:         sync.TransformContacts(patient),
			UpdateContacts:   true,
			EntityInfo:       patientEntity.Info,
			UpdateEntityInfo: true,
		})
		if err != nil {
			return errors.Errorf("Unable to update patient information for %s : %s ", patient.ID, err)
		}

		// update corrresponding threads
		threadsForMembersRes, err := s.threading.ThreadsForMember(ctx, &threading.ThreadsForMemberRequest{
			PrimaryOnly: true,
			EntityID:    patientEntity.ID,
		})
		if err != nil {
			return errors.Errorf("Unable to get threads for members for %s : %s", patient.ID, err)
		}

		// update the system title for threads
		for _, thread := range threadsForMembersRes.Threads {
			if _, err := s.threading.UpdateThread(ctx, &threading.UpdateThreadRequest{
				ActorEntityID: thread.OrganizationID,
				ThreadID:      thread.ID,
				SystemTitle:   updateRes.Entity.Info.DisplayName,
			}); err != nil {
				return errors.Errorf("Unable to update system title for thread %s : %s", thread.ID, err)
			}
		}

		golog.Infof("patient update in Hint (%s) triggered an update in Spruce (%s)", patient.ID, patientEntity.ID)
	}

	return nil
}

func (s *syncEvent) processPatientAddEvent(ctx context.Context, cfg *sync.Config, event *sync.Event) error {
	addEvent := event.GetPatientAddEvent()
	if addEvent == nil {
		return errors.Errorf("Expected add event for %s and org %s but got none", event.Type.String(), event.OrganizationEntityID)
	}

	// go through the list of patients and create the appropriate threads for each
	for _, patient := range addEvent.GetPatients() {
		sanitizePatient(patient)
		if err := s.createPatientAndThread(ctx, patient, cfg, event); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (s *syncEvent) createPatientAndThread(ctx context.Context, patient *sync.Patient, cfg *sync.Config, event *sync.Event) error {

	var threadType threading.ThreadType
	orgID := cfg.OrganizationEntityID
	source := event.Source
	switch cfg.ThreadCreationType {
	case sync.THREAD_CREATION_TYPE_STANDARD:
		if len(patient.PhoneNumbers) == 0 && len(patient.EmailAddresses) == 0 {
			golog.Warningf("Ignoring patient %s since we don't have at least one valid phone number and email address for the patient", patient.ID)
			return nil
		}
		threadType = threading.THREAD_TYPE_EXTERNAL
	case sync.THREAD_CREATION_TYPE_SECURE:

		var requirePhoneAndEmailForSecureConversationCreation bool
		val, err := settings.GetBooleanValue(ctx, s.settings, &settings.GetValuesRequest{
			NodeID: orgID,
			Keys: []*settings.ConfigKey{
				{
					Key: invite.ConfigKeyTwoFactorVerificationForSecureConversation,
				},
			},
		})
		if err != nil {
			golog.Errorf("Unable to query for setting for org %s: %s", orgID, err)
		} else {
			requirePhoneAndEmailForSecureConversationCreation = val.Value
		}

		if requirePhoneAndEmailForSecureConversationCreation {
			// ensure that we have at least one phone number and email address before proceeding
			if len(patient.PhoneNumbers) == 0 || len(patient.EmailAddresses) == 0 {
				golog.Warningf("Ignoring patient %s since we dont have at least one valid phone number and email address for the patient", patient.ID)
				return nil
			}
		} else if len(patient.PhoneNumbers) == 0 && len(patient.EmailAddresses) == 0 {
			golog.Warningf("Ignoring patient %s since we dont have at least one valid phone number and email address for the patient", patient.ID)
			return nil
		}

		threadType = threading.THREAD_TYPE_SECURE_EXTERNAL
	default:
		return errors.Errorf("unknown thread creation type for org %s", event.OrganizationEntityID)
	}

	var invitePatient bool
	switch ev := event.Event.(type) {
	case *sync.Event_PatientAddEvent:
		invitePatient = ev.PatientAddEvent.AutoInvitePatients && threadType == threading.THREAD_TYPE_SECURE_EXTERNAL
	case *sync.Event_PatientUpdateEvent:
		invitePatient = ev.PatientUpdateEvent.AutoInvitePatients && threadType == threading.THREAD_TYPE_SECURE_EXTERNAL
	}

	// check if the patient already exists as an entity based on the patient ID
	if patient.ID == "" {
		return errors.Errorf("cannot create patient with no external ID for org %s", orgID)
	}

	entityType := directory.EntityType_EXTERNAL
	if threadType == threading.THREAD_TYPE_SECURE_EXTERNAL {
		entityType = directory.EntityType_PATIENT
	}

	res, err := s.directory.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_ExternalID{
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
	if res != nil {
		for _, entity := range res.Entities {
			if entity.Status == directory.EntityStatus_DELETED {
				// if the entity has been deleted, ignore the sync
				return nil
			} else if entity.Status == directory.EntityStatus_ACTIVE {
				externalEntity = entity
				break
			}
		}
	}

	if externalEntity == nil {
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
			Contacts: sync.TransformContacts(patient),
			EntityInfo: &directory.EntityInfo{
				FirstName: patient.FirstName,
				LastName:  patient.LastName,
				DOB:       sync.TransformDOB(patient.DOB),
				Gender:    sync.TransformGender(patient.Gender),
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

		memberEntityIDs := make([]string, 0, 2)
		memberEntityIDs = append(memberEntityIDs, orgID)

		if threadType == threading.THREAD_TYPE_SECURE_EXTERNAL {
			memberEntityIDs = append(memberEntityIDs, externalEntity.ID)
		}

		// create thread
		createThreadRes, err := s.threading.CreateEmptyThread(ctx, &threading.CreateEmptyThreadRequest{
			OrganizationID:  orgID,
			PrimaryEntityID: externalEntity.ID,
			MemberEntityIDs: memberEntityIDs,
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

	if invitePatient {

		var phoneNumber string
		if len(patient.PhoneNumbers) > 0 {
			phoneNumber = patient.PhoneNumbers[0].Number
		}
		var email string
		if len(patient.EmailAddresses) > 0 {
			email = patient.EmailAddresses[0]
		}

		if _, err := s.invite.InvitePatients(ctx, &invite.InvitePatientsRequest{
			OrganizationEntityID: orgID,
			Patients: []*invite.Patient{
				{
					FirstName:      externalEntity.Info.FirstName,
					PhoneNumber:    phoneNumber,
					Email:          email,
					ParkedEntityID: externalEntity.ID,
				},
			},
		}); err != nil {
			golog.Errorf("Unable to invite patient %s : %s", externalEntity.ID, err)
		}
	}

	// TODO: link patient back to source (probably by posting another notification that creation is complete)
	fmt.Println(deeplink.ThreadURLShareable(s.webDomain, orgID, thread.ID))

	return nil
}
