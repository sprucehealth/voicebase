package blackbox

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/blackbox/harness"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Tests contains the test methods for the auth service
type tests struct{}

// NewTests returns an initialized instance of tests
func NewTests() harness.TestSuite {
	return &tests{}
}

// SuiteName returns the name of this test suite
func (t *tests) SuiteName() string {
	return "DirectoryService"
}

// GeneratePayload conforms to the BBTest harness payload generation
func (t *tests) GeneratePayload() interface{} {
	conn, err := grpc.Dial(harness.GetConfig("directory_service_endpoint"), grpc.WithInsecure(), grpc.WithTimeout(2*time.Second))
	if err != nil {
		golog.Fatalf("Unable to dial grpc server: %s", err)
	}
	client := directory.NewDirectoryClient(conn)
	return client
}

var (
	maxEntityNameSize          int64 = 250
	maxExternalEntityIDSize    int64 = 100
	maxContactsPerEntity       int64 = 20
	maxRequestedInfoDepth      int64 = 4
	maxRequestedInfoDimensions int64 = 7
	maxEntityInfoSize          int64 = 250
)

func randomEntityType() directory.EntityType {
	var types []directory.EntityType
	for _, v := range directory.EntityType_value {
		types = append(types, directory.EntityType(v))
	}
	return types[harness.RandInt64N(int64(len(types)))]
}

func randomContactType() directory.ContactType {
	var types []directory.ContactType
	for _, v := range directory.ContactType_value {
		types = append(types, directory.ContactType(v))
	}
	return types[harness.RandInt64N(int64(len(types)))]
}

func randomEntityInformation() directory.EntityInformation {
	var types []directory.EntityInformation
	for _, v := range directory.EntityInformation_value {
		types = append(types, directory.EntityInformation(v))
	}
	return types[harness.RandInt64N(int64(len(types)))]
}

func optionalRandomRequestedInformation() *directory.RequestedInformation {
	if harness.RandBool() {
		return nil
	}
	ri := &directory.RequestedInformation{
		Depth: harness.RandInt64N(maxRequestedInfoDepth),
	}
	dims := harness.RandInt64N(maxRequestedInfoDimensions)
	for i := int64(0); i < dims; i++ {
		ri.EntityInformation = append(ri.EntityInformation, randomEntityInformation())
	}
	return ri
}

func optionalRandomExternalID() string {
	if harness.RandBool() {
		return ""
	}
	return harness.RandLengthString(maxExternalEntityIDSize)
}

func optionalRandomContactsSlice() []*directory.Contact {
	var contacts []*directory.Contact
	if harness.RandBool() {
		for i := int64(0); i < harness.RandInt64N(maxContactsPerEntity); i++ {
			contacts = append(contacts, randomContact())
		}
	}
	return contacts
}

func randomContact() *directory.Contact {
	contactType := randomContactType()
	var contactValue string
	switch contactType {
	case directory.ContactType_PHONE:
		contactValue = harness.RandPhoneNumber()
	case directory.ContactType_EMAIL:
		contactValue = harness.RandEmail()
	default:
		harness.Failf("Unknown contact type %v", contactType)
	}
	return &directory.Contact{
		ContactType: contactType,
		Value:       contactValue,
		Provisioned: harness.RandBool(),
	}
}

func assertEntity(e *directory.Entity) {
	harness.Assert(e.ID != "")
	harness.Assert(e.Info != nil)
}

func randomValidCreateEntityRequest(initialMembershipEntityID string) *directory.CreateEntityRequest {
	return &directory.CreateEntityRequest{
		EntityInfo:                randomEntityInfo(),
		Type:                      randomEntityType(),
		ExternalID:                optionalRandomExternalID(),
		InitialMembershipEntityID: initialMembershipEntityID,
		Contacts:                  optionalRandomContactsSlice(),
		RequestedInformation:      optionalRandomRequestedInformation(),
	}
}

func randomEntityInfo() *directory.EntityInfo {
	return &directory.EntityInfo{
		FirstName:     harness.RandLengthString(maxEntityInfoSize),
		MiddleInitial: harness.RandLengthString(1),
		LastName:      harness.RandLengthString(maxEntityInfoSize),
		GroupName:     harness.RandLengthString(maxEntityInfoSize),
		DisplayName:   harness.RandLengthString(maxEntityInfoSize),
		Note:          harness.RandLengthString(maxEntityInfoSize),
	}
}

func createEntity(client directory.DirectoryClient, req *directory.CreateEntityRequest) (*directory.CreateEntityRequest, *directory.CreateEntityResponse) {
	var err error
	var resp *directory.CreateEntityResponse
	golog.Debugf("CreateEntity call: %+v", req)
	harness.Profile("DirectoryService:CreateEntity", func() { resp, err = client.CreateEntity(context.Background(), req) })
	golog.Debugf("CreateEntity response: %+v", resp)
	harness.FailErr(err)

	assertEntity(resp.GetEntity())
	return req, resp
}

func createMembership(client directory.DirectoryClient, entityID, targetEntityID string, requestedInformation *directory.RequestedInformation) (*directory.CreateMembershipRequest, *directory.CreateMembershipResponse) {
	var err error
	req := &directory.CreateMembershipRequest{
		EntityID:             entityID,
		TargetEntityID:       targetEntityID,
		RequestedInformation: requestedInformation,
	}
	var resp *directory.CreateMembershipResponse
	golog.Debugf("CreateMembership call: %+v", req)
	harness.Profile("DirectoryService:CreateMembership", func() { resp, err = client.CreateMembership(context.Background(), req) })
	golog.Debugf("CreateMembership response: %+v", resp)
	harness.FailErr(err)

	assertEntity(resp.GetEntity())
	return req, resp
}

func lookupEntitiesByIDRequest(entityID string, requestedInformation *directory.RequestedInformation) *directory.LookupEntitiesRequest {
	return &directory.LookupEntitiesRequest{
		LookupKeyType:        directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof:       &directory.LookupEntitiesRequest_EntityID{EntityID: entityID},
		RequestedInformation: requestedInformation,
	}
}

func lookupEntitiesByExternalIDRequest(externalID string, requestedInformation *directory.RequestedInformation) *directory.LookupEntitiesRequest {
	return &directory.LookupEntitiesRequest{
		LookupKeyType:        directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof:       &directory.LookupEntitiesRequest_ExternalID{ExternalID: externalID},
		RequestedInformation: requestedInformation,
	}
}

func lookupEntities(client directory.DirectoryClient, req *directory.LookupEntitiesRequest) (*directory.LookupEntitiesRequest, *directory.LookupEntitiesResponse) {
	var err error
	var resp *directory.LookupEntitiesResponse
	golog.Debugf("LookupEntities call: %+v", req)
	harness.Profile("DirectoryService:LookupEntities", func() { resp, err = client.LookupEntities(context.Background(), req) })
	golog.Debugf("LookupEntities response: %+v", resp)
	harness.FailErr(err)

	harness.Assert(len(resp.GetEntities()) > 0)
	return req, resp
}

func createContact(client directory.DirectoryClient, req *directory.CreateContactRequest) (*directory.CreateContactRequest, *directory.CreateContactResponse) {
	var err error
	var resp *directory.CreateContactResponse
	golog.Debugf("CreateContact call: %+v", req)
	harness.Profile("DirectoryService:CreateContact", func() { resp, err = client.CreateContact(context.Background(), req) })
	golog.Debugf("CreateContact response: %+v", resp)
	harness.FailErr(err)

	assertEntity(resp.Entity)
	return req, resp
}

func lookupEntitiesByContact(client directory.DirectoryClient, req *directory.LookupEntitiesByContactRequest) (*directory.LookupEntitiesByContactRequest, *directory.LookupEntitiesByContactResponse) {
	var err error
	var resp *directory.LookupEntitiesByContactResponse
	golog.Debugf("LookupEntitiesByContact call: %+v", req)
	harness.Profile("DirectoryService:LookupEntitiesByContact", func() { resp, err = client.LookupEntitiesByContact(context.Background(), req) })
	golog.Debugf("LookupEntitiesByContact response: %+v", resp)
	harness.FailErr(err)

	harness.Assert(len(resp.GetEntities()) > 0)
	return req, resp
}

func (t *tests) BBTestGRPCBasicEntityMembershipCreation(client interface{}) {
	directoryClient, ok := client.(directory.DirectoryClient)
	if !ok {
		harness.Failf("Unable to unpack client: %+v", client)
	}
	// Create a random entity
	createEntityReq, createEntityResp := createEntity(directoryClient, randomValidCreateEntityRequest(""))

	// Create another random entity and make him an initial member of the firse one
	targetMembershipID := createEntityResp.Entity.ID
	createEntityReq = randomValidCreateEntityRequest(targetMembershipID)
	createEntityReq.RequestedInformation = &directory.RequestedInformation{
		Depth:             1,
		EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
	}
	_, createEntityResp = createEntity(directoryClient, createEntityReq)
	harness.Assert(len(createEntityResp.Entity.Memberships) == 1, fmt.Sprintf("Expected only 1 membership but got %+v", createEntityResp.Entity.Memberships))
	harness.Assert(createEntityResp.Entity.GetMemberships()[0].ID == targetMembershipID)

	// Create another random entity
	createEntityReq, createEntityResp = createEntity(directoryClient, randomValidCreateEntityRequest(""))

	// Directly create a membership
	_, createMembershipResp := createMembership(directoryClient, createEntityResp.GetEntity().ID, targetMembershipID, &directory.RequestedInformation{
		Depth:             1,
		EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
	})
	harness.Assert(len(createMembershipResp.Entity.Memberships) == 1)
	harness.Assert(createMembershipResp.Entity.GetMemberships()[0].ID == targetMembershipID)
}

func (t *tests) BBTestGRPCBasicEntityLookup(client interface{}) {
	directoryClient, ok := client.(directory.DirectoryClient)
	if !ok {
		harness.Failf("Unable to unpack client: %+v", client)
	}
	// Create a random entity
	requestedInfo := &directory.RequestedInformation{
		Depth:             1,
		EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
	}
	createEntityReq := randomValidCreateEntityRequest("")
	createEntityReq.RequestedInformation = requestedInfo
	createEntityReq, createEntityResp := createEntity(directoryClient, createEntityReq)

	// Lookup the entity
	_, lookupEntitiesResp := lookupEntities(directoryClient, lookupEntitiesByIDRequest(createEntityResp.Entity.ID, requestedInfo))
	harness.Assert(len(lookupEntitiesResp.Entities) == 1)
	harness.AssertEqual(createEntityResp.Entity, lookupEntitiesResp.Entities[0])
}

func (t *tests) BBTestGRPCContactManagement(client interface{}) {
	directoryClient, ok := client.(directory.DirectoryClient)
	if !ok {
		harness.Failf("Unable to unpack client: %+v", client)
	}
	// Create a random entity and membership
	createEntityReq, createEntityResp := createEntity(directoryClient, randomValidCreateEntityRequest(""))
	createEntityReq, createEntityResp = createEntity(directoryClient, randomValidCreateEntityRequest(createEntityResp.Entity.ID))

	// Add a contact for the entity
	contact := randomContact()
	_, createContactResp := createContact(directoryClient, &directory.CreateContactRequest{
		Contact:              contact,
		EntityID:             createEntityResp.Entity.ID,
		RequestedInformation: createEntityReq.RequestedInformation,
	})
	harness.AssertEqual(createEntityResp.Entity.ID, createContactResp.Entity.ID)

	// Look the entity up by contact
	_, lookupEntitiesByContactResp := lookupEntitiesByContact(directoryClient, &directory.LookupEntitiesByContactRequest{
		ContactValue:         contact.Value,
		RequestedInformation: createEntityReq.RequestedInformation,
	})
	harness.Assert(len(lookupEntitiesByContactResp.Entities) == 1)
	harness.AssertEqual(createContactResp.Entity, lookupEntitiesByContactResp.Entities[0])

	// Create a random entity and membership
	createEntityReq, createEntityResp = createEntity(directoryClient, randomValidCreateEntityRequest(""))
	createEntityReq, createEntityResp = createEntity(directoryClient, randomValidCreateEntityRequest(createEntityResp.Entity.ID))

	// Add the original contact to the entity
	_, createContactResp = createContact(directoryClient, &directory.CreateContactRequest{
		Contact:  contact,
		EntityID: createEntityResp.Entity.ID,
		RequestedInformation: &directory.RequestedInformation{
			Depth:             2,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
	})
	harness.AssertEqual(createEntityResp.Entity.ID, createContactResp.Entity.ID)

	// Look the entity up by contact
	_, lookupEntitiesByContactResp = lookupEntitiesByContact(directoryClient, &directory.LookupEntitiesByContactRequest{
		ContactValue: contact.Value,
		RequestedInformation: &directory.RequestedInformation{
			Depth:             2,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS, directory.EntityInformation_MEMBERSHIPS},
		},
	})
	harness.Assert(len(lookupEntitiesByContactResp.Entities) == 2)
	for i, v := range lookupEntitiesByContactResp.Entities {
		harness.AssertEqual(1, len(v.Memberships), fmt.Sprintf("Element %d in %v", i, lookupEntitiesByContactResp.Entities))
	}
}
