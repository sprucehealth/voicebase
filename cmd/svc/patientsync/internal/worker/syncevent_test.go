package worker

import (
	"context"
	"encoding/base64"
	"testing"

	dalmock "github.com/sprucehealth/backend/cmd/svc/patientsync/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/patientsync/internal/sync"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	directorymock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/threading"
	threadingmock "github.com/sprucehealth/backend/svc/threading/mock"
)

func TestStandardThreadSync(t *testing.T) {
	orgID := "orgID1"
	event := sync.Event{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		Type:                 sync.EVENT_TYPE_PATIENT_ADD,
		Event: &sync.Event_PatientAddEvent{
			PatientAddEvent: &sync.PatientAddEvent{
				Patients: []*sync.Patient{
					{
						ID:        "12345",
						FirstName: "FirstName1",
						LastName:  "LastName1",
						PhoneNumbers: []*sync.Phone{
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+12222222222",
							},
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+13333333333",
							},
						},
						EmailAddresses: []string{"test@example.com", "test2@example.com"},
					},
				},
			},
		},
	}

	tmock := threadingmock.New(t)
	dirmock := directorymock.New(t)
	dmock := dalmock.New(t)
	defer mock.FinishAll(dmock, tmock, dirmock)

	dmock.Expect(mock.NewExpectation(dmock.SyncConfigForOrg, orgID, "SOURCE_ELATION").WithReturns(&sync.Config{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		ThreadCreationType:   sync.THREAD_CREATION_TYPE_STANDARD,
	}, nil))

	dirmock.Expect(mock.NewExpectation(dirmock.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "elation_12345",
		},
		MemberOfEntity: orgID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}))

	patientContacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12222222222",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+13333333333",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test@example.com",
		},

		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test2@example.com",
		},
	}

	dirmock.Expect(mock.NewExpectation(dirmock.CreateEntity, &directory.CreateEntityRequest{
		Type:                      directory.EntityType_EXTERNAL,
		ExternalID:                "elation_12345",
		InitialMembershipEntityID: orgID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		Contacts: patientContacts,
		EntityInfo: &directory.EntityInfo{
			FirstName: "FirstName1",
			LastName:  "LastName1",
		},
	}).WithReturns(&directory.CreateEntityResponse{
		Entity: &directory.Entity{
			ID: "ent_1",
			Info: &directory.EntityInfo{
				FirstName:   "FirstName1",
				LastName:    "LastName1",
				DisplayName: "DisplayName1",
			},
			Contacts: patientContacts,
			Memberships: []*directory.Entity{
				{
					ID:   orgID,
					Type: directory.EntityType_ORGANIZATION,
				},
			},
		},
	}, nil))

	tmock.Expect(mock.NewExpectation(tmock.ThreadsForMember, &threading.ThreadsForMemberRequest{
		PrimaryOnly: true,
		EntityID:    "ent_1",
	}))
	tmock.Expect(mock.NewExpectation(tmock.CreateEmptyThread, &threading.CreateEmptyThreadRequest{
		OrganizationID:  orgID,
		PrimaryEntityID: "ent_1",
		MemberEntityIDs: []string{orgID},
		Type:            threading.THREAD_TYPE_EXTERNAL,
		Summary:         "DisplayName1",
		SystemTitle:     "DisplayName1",
		Origin:          threading.THREAD_ORIGIN_SYNC,
	}).WithReturns(&threading.CreateEmptyThreadResponse{
		Thread: &threading.Thread{
			ID:             "thread_1",
			OrganizationID: orgID,
		},
	}, nil))

	data, err := event.Marshal()
	test.OK(t, err)
	s := NewSyncEvent(dmock, dirmock, tmock, nil, "", "")
	test.OK(t, s.(*syncEvent).processSyncEvent(context.Background(), base64.StdEncoding.EncodeToString(data)))
}

func TestStandardThreadSync_EntityExists(t *testing.T) {
	orgID := "orgID1"
	event := sync.Event{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		Type:                 sync.EVENT_TYPE_PATIENT_ADD,
		Event: &sync.Event_PatientAddEvent{
			PatientAddEvent: &sync.PatientAddEvent{
				Patients: []*sync.Patient{
					{
						ID:        "12345",
						FirstName: "FirstName1",
						LastName:  "LastName1",
						PhoneNumbers: []*sync.Phone{
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+12222222222",
							},
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+13333333333",
							},
						},
						EmailAddresses: []string{"test@example.com", "test2@example.com"},
					},
				},
			},
		},
	}

	tmock := threadingmock.New(t)
	dirmock := directorymock.New(t)
	dmock := dalmock.New(t)
	defer mock.FinishAll(dmock, tmock, dirmock)

	dmock.Expect(mock.NewExpectation(dmock.SyncConfigForOrg, orgID, "SOURCE_ELATION").WithReturns(&sync.Config{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		ThreadCreationType:   sync.THREAD_CREATION_TYPE_STANDARD,
	}, nil))

	patientContacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12222222222",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+13333333333",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test@example.com",
		},

		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test2@example.com",
		},
	}

	dirmock.Expect(mock.NewExpectation(dirmock.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "elation_12345",
		},
		MemberOfEntity: orgID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				Status: directory.EntityStatus_ACTIVE,
				ID:     "ent_1",
				Info: &directory.EntityInfo{
					FirstName:   "FirstName1",
					LastName:    "LastName1",
					DisplayName: "DisplayName1",
				},
				Contacts: patientContacts,
				Memberships: []*directory.Entity{
					{
						ID:   orgID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	tmock.Expect(mock.NewExpectation(tmock.ThreadsForMember, &threading.ThreadsForMemberRequest{
		PrimaryOnly: true,
		EntityID:    "ent_1",
	}))
	tmock.Expect(mock.NewExpectation(tmock.CreateEmptyThread, &threading.CreateEmptyThreadRequest{
		OrganizationID:  orgID,
		PrimaryEntityID: "ent_1",
		MemberEntityIDs: []string{orgID},
		Type:            threading.THREAD_TYPE_EXTERNAL,
		Summary:         "DisplayName1",
		SystemTitle:     "DisplayName1",
		Origin:          threading.THREAD_ORIGIN_SYNC,
	}).WithReturns(&threading.CreateEmptyThreadResponse{
		Thread: &threading.Thread{
			ID:             "thread_1",
			OrganizationID: orgID,
		},
	}, nil))

	data, err := event.Marshal()
	test.OK(t, err)
	s := NewSyncEvent(dmock, dirmock, tmock, nil, "", "")
	test.OK(t, s.(*syncEvent).processSyncEvent(context.Background(), base64.StdEncoding.EncodeToString(data)))
}

func TestStandardThreadSync_ThreadExists(t *testing.T) {
	orgID := "orgID1"
	event := sync.Event{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		Type:                 sync.EVENT_TYPE_PATIENT_ADD,
		Event: &sync.Event_PatientAddEvent{
			PatientAddEvent: &sync.PatientAddEvent{
				Patients: []*sync.Patient{
					{
						ID:        "12345",
						FirstName: "FirstName1",
						LastName:  "LastName1",
						PhoneNumbers: []*sync.Phone{
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+12222222222",
							},
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+13333333333",
							},
						},
						EmailAddresses: []string{"test@example.com", "test2@example.com"},
					},
				},
			},
		},
	}

	tmock := threadingmock.New(t)
	dirmock := directorymock.New(t)
	dmock := dalmock.New(t)
	defer mock.FinishAll(dmock, tmock, dirmock)

	dmock.Expect(mock.NewExpectation(dmock.SyncConfigForOrg, orgID, "SOURCE_ELATION").WithReturns(&sync.Config{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		ThreadCreationType:   sync.THREAD_CREATION_TYPE_STANDARD,
	}, nil))

	patientContacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12222222222",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+13333333333",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test@example.com",
		},

		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test2@example.com",
		},
	}

	dirmock.Expect(mock.NewExpectation(dirmock.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "elation_12345",
		},
		MemberOfEntity: orgID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				Status: directory.EntityStatus_ACTIVE,
				ID:     "ent_1",
				Info: &directory.EntityInfo{
					FirstName:   "FirstName1",
					LastName:    "LastName1",
					DisplayName: "DisplayName1",
				},
				Contacts: patientContacts,
				Memberships: []*directory.Entity{
					{
						ID:   orgID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	tmock.Expect(mock.NewExpectation(tmock.ThreadsForMember, &threading.ThreadsForMemberRequest{
		PrimaryOnly: true,
		EntityID:    "ent_1",
	}).WithReturns(&threading.ThreadsForMemberResponse{
		Threads: []*threading.Thread{
			{
				ID:             "thread_1",
				OrganizationID: orgID,
			},
		},
	}, nil))

	data, err := event.Marshal()
	test.OK(t, err)
	s := NewSyncEvent(dmock, dirmock, tmock, nil, "", "")
	test.OK(t, s.(*syncEvent).processSyncEvent(context.Background(), base64.StdEncoding.EncodeToString(data)))
}

func TestStandardThreadSync_Update_NoEntity(t *testing.T) {
	orgID := "orgID1"
	event := sync.Event{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		Event: &sync.Event_PatientUpdateEvent{
			PatientUpdateEvent: &sync.PatientUpdatedEvent{
				Patients: []*sync.Patient{
					{
						ID:        "12345",
						FirstName: "FirstName1",
						LastName:  "LastName1",
						PhoneNumbers: []*sync.Phone{
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+12222222222",
							},
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+13333333333",
							},
						},
						EmailAddresses: []string{"test@example.com", "test2@example.com"},
					},
				},
			},
		},
	}

	tmock := threadingmock.New(t)
	dirmock := directorymock.New(t)
	dmock := dalmock.New(t)
	defer mock.FinishAll(dmock, tmock, dirmock)

	dmock.Expect(mock.NewExpectation(dmock.SyncConfigForOrg, orgID, "SOURCE_ELATION").WithReturns(&sync.Config{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		ThreadCreationType:   sync.THREAD_CREATION_TYPE_STANDARD,
	}, nil))

	dirmock.Expect(mock.NewExpectation(dirmock.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "elation_12345",
		},
		MemberOfEntity: orgID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_PATIENT, directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}))

	dirmock.Expect(mock.NewExpectation(dirmock.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "elation_12345",
		},
		MemberOfEntity: orgID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}))

	patientContacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12222222222",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+13333333333",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test@example.com",
		},

		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test2@example.com",
		},
	}

	dirmock.Expect(mock.NewExpectation(dirmock.CreateEntity, &directory.CreateEntityRequest{
		Type:                      directory.EntityType_EXTERNAL,
		ExternalID:                "elation_12345",
		InitialMembershipEntityID: orgID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		Contacts: patientContacts,
		EntityInfo: &directory.EntityInfo{
			FirstName: "FirstName1",
			LastName:  "LastName1",
		},
	}).WithReturns(&directory.CreateEntityResponse{
		Entity: &directory.Entity{
			ID: "ent_1",
			Info: &directory.EntityInfo{
				FirstName:   "FirstName1",
				LastName:    "LastName1",
				DisplayName: "DisplayName1",
			},
			Contacts: patientContacts,
			Memberships: []*directory.Entity{
				{
					ID:   orgID,
					Type: directory.EntityType_ORGANIZATION,
				},
			},
		},
	}, nil))

	tmock.Expect(mock.NewExpectation(tmock.ThreadsForMember, &threading.ThreadsForMemberRequest{
		PrimaryOnly: true,
		EntityID:    "ent_1",
	}))
	tmock.Expect(mock.NewExpectation(tmock.CreateEmptyThread, &threading.CreateEmptyThreadRequest{
		OrganizationID:  orgID,
		PrimaryEntityID: "ent_1",
		MemberEntityIDs: []string{orgID},
		Type:            threading.THREAD_TYPE_EXTERNAL,
		Summary:         "DisplayName1",
		SystemTitle:     "DisplayName1",
		Origin:          threading.THREAD_ORIGIN_SYNC,
	}).WithReturns(&threading.CreateEmptyThreadResponse{
		Thread: &threading.Thread{
			ID:             "thread_1",
			OrganizationID: orgID,
		},
	}, nil))

	data, err := event.Marshal()
	test.OK(t, err)
	s := NewSyncEvent(dmock, dirmock, tmock, nil, "", "")
	test.OK(t, s.(*syncEvent).processSyncEvent(context.Background(), base64.StdEncoding.EncodeToString(data)))
}

func TestStandardThreadSync_Update_EntityExists_NoDifference(t *testing.T) {
	orgID := "orgID1"
	event := sync.Event{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		Event: &sync.Event_PatientUpdateEvent{
			PatientUpdateEvent: &sync.PatientUpdatedEvent{
				Patients: []*sync.Patient{
					{
						ID:        "12345",
						FirstName: "FirstName1",
						LastName:  "LastName1",
						PhoneNumbers: []*sync.Phone{
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+12222222222",
							},
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+13333333333",
							},
						},
						EmailAddresses: []string{"test@example.com", "test2@example.com"},
					},
				},
			},
		},
	}

	tmock := threadingmock.New(t)
	dirmock := directorymock.New(t)
	dmock := dalmock.New(t)
	defer mock.FinishAll(dmock, tmock, dirmock)

	dmock.Expect(mock.NewExpectation(dmock.SyncConfigForOrg, orgID, "SOURCE_ELATION").WithReturns(&sync.Config{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		ThreadCreationType:   sync.THREAD_CREATION_TYPE_STANDARD,
	}, nil))

	patientContacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12222222222",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+13333333333",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test@example.com",
		},

		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test2@example.com",
		},
	}

	dirmock.Expect(mock.NewExpectation(dirmock.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "elation_12345",
		},
		MemberOfEntity: orgID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_PATIENT, directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: "ent_1",
				Info: &directory.EntityInfo{
					FirstName:   "FirstName1",
					LastName:    "LastName1",
					DisplayName: "DisplayName1",
				},
				Status:   directory.EntityStatus_ACTIVE,
				Contacts: patientContacts,
				Memberships: []*directory.Entity{
					{
						ID:   orgID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	data, err := event.Marshal()
	test.OK(t, err)
	s := NewSyncEvent(dmock, dirmock, tmock, nil, "", "")
	test.OK(t, s.(*syncEvent).processSyncEvent(context.Background(), base64.StdEncoding.EncodeToString(data)))
}

func TestStandardThreadSync_Update_EntityExists_Deleted(t *testing.T) {
	orgID := "orgID1"
	event := sync.Event{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		Event: &sync.Event_PatientUpdateEvent{
			PatientUpdateEvent: &sync.PatientUpdatedEvent{
				Patients: []*sync.Patient{
					{
						ID:        "12345",
						FirstName: "FirstName1",
						LastName:  "LastName1",
						PhoneNumbers: []*sync.Phone{
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+12222222222",
							},
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+13333333333",
							},
						},
						EmailAddresses: []string{"test@example.com", "test2@example.com"},
					},
				},
			},
		},
	}

	tmock := threadingmock.New(t)
	dirmock := directorymock.New(t)
	dmock := dalmock.New(t)
	defer mock.FinishAll(dmock, tmock, dirmock)

	dmock.Expect(mock.NewExpectation(dmock.SyncConfigForOrg, orgID, "SOURCE_ELATION").WithReturns(&sync.Config{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		ThreadCreationType:   sync.THREAD_CREATION_TYPE_STANDARD,
	}, nil))

	patientContacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12222222222",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+13333333333",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test@example.com",
		},

		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test2@example.com",
		},
	}

	dirmock.Expect(mock.NewExpectation(dirmock.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "elation_12345",
		},
		MemberOfEntity: orgID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_PATIENT, directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: "ent_1",
				Info: &directory.EntityInfo{
					FirstName:   "FirstName1",
					LastName:    "LastName1",
					DisplayName: "DisplayName1",
				},
				Status:   directory.EntityStatus_DELETED,
				Contacts: patientContacts,
				Memberships: []*directory.Entity{
					{
						ID:   orgID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	data, err := event.Marshal()
	test.OK(t, err)
	s := NewSyncEvent(dmock, dirmock, tmock, nil, "", "")
	test.OK(t, s.(*syncEvent).processSyncEvent(context.Background(), base64.StdEncoding.EncodeToString(data)))
}

func TestStandardThreadSync_Update_EntityExists_Differs(t *testing.T) {
	orgID := "orgID1"
	event := sync.Event{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		Event: &sync.Event_PatientUpdateEvent{
			PatientUpdateEvent: &sync.PatientUpdatedEvent{
				Patients: []*sync.Patient{
					{
						ID:        "12345",
						FirstName: "FirstName1",
						LastName:  "LastName1",
						PhoneNumbers: []*sync.Phone{
							{
								Type:   sync.PHONE_TYPE_MOBILE,
								Number: "+12222222222",
							},
						},
						EmailAddresses: []string{"test@example.com", "test2@example.com"},
					},
				},
			},
		},
	}

	tmock := threadingmock.New(t)
	dirmock := directorymock.New(t)
	dmock := dalmock.New(t)
	defer mock.FinishAll(dmock, tmock, dirmock)

	dmock.Expect(mock.NewExpectation(dmock.SyncConfigForOrg, orgID, "SOURCE_ELATION").WithReturns(&sync.Config{
		OrganizationEntityID: orgID,
		Source:               sync.SOURCE_ELATION,
		ThreadCreationType:   sync.THREAD_CREATION_TYPE_STANDARD,
	}, nil))

	patientContacts := []*directory.Contact{
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12222222222",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_PHONE,
			Value:       "+13333333333",
			Label:       "Mobile",
		},
		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test@example.com",
		},

		{
			ContactType: directory.ContactType_EMAIL,
			Value:       "test2@example.com",
		},
	}

	dirmock.Expect(mock.NewExpectation(dirmock.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: "elation_12345",
		},
		MemberOfEntity: orgID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_PATIENT, directory.EntityType_EXTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: "ent_1",
				Info: &directory.EntityInfo{
					FirstName:   "FirstName1",
					LastName:    "LastName1",
					DisplayName: "DisplayName1",
				},
				Status:   directory.EntityStatus_ACTIVE,
				Contacts: patientContacts,
				Memberships: []*directory.Entity{
					{
						ID:   orgID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	// update entity
	dirmock.Expect(mock.NewExpectation(dirmock.UpdateEntity, &directory.UpdateEntityRequest{
		EntityID:       "ent_1",
		UpdateContacts: true,
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+12222222222",
				Label:       "Mobile",
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "test@example.com",
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "test2@example.com",
			},
		},
		UpdateEntityInfo: true,
		EntityInfo: &directory.EntityInfo{
			FirstName:   "FirstName1",
			LastName:    "LastName1",
			DisplayName: "DisplayName1",
		},
	}).WithReturns(&directory.UpdateEntityResponse{
		Entity: &directory.Entity{
			ID: "ent_1",
			Contacts: []*directory.Contact{
				{
					ContactType: directory.ContactType_PHONE,
					Value:       "+12222222222",
					Label:       "Mobile",
				},
				{
					ContactType: directory.ContactType_EMAIL,
					Value:       "test@example.com",
				},
				{
					ContactType: directory.ContactType_EMAIL,
					Value:       "test2@example.com",
				},
			},
			Info: &directory.EntityInfo{
				FirstName:   "FirstName1",
				LastName:    "LastName1",
				DisplayName: "DisplayName1",
			},
		},
	}, nil))

	// update thread
	tmock.Expect(mock.NewExpectation(tmock.ThreadsForMember, &threading.ThreadsForMemberRequest{
		PrimaryOnly: true,
		EntityID:    "ent_1",
	}).WithReturns(&threading.ThreadsForMemberResponse{
		Threads: []*threading.Thread{
			{
				ID:             "thread_1",
				OrganizationID: orgID,
			},
		},
	}, nil))

	tmock.Expect(mock.NewExpectation(tmock.UpdateThread, &threading.UpdateThreadRequest{
		ActorEntityID: orgID,
		SystemTitle:   "DisplayName1",
		ThreadID:      "thread_1",
	}))

	data, err := event.Marshal()
	test.OK(t, err)
	s := NewSyncEvent(dmock, dirmock, tmock, nil, "", "")
	test.OK(t, s.(*syncEvent).processSyncEvent(context.Background(), base64.StdEncoding.EncodeToString(data)))
}
