package main

import (
	"context"
	"encoding/csv"
	"flag"
	"io"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
)

type cleanupSpamAccountsCmd struct {
	cnf          *config
	directoryCLI directory.DirectoryClient
	excommsCLI   excomms.ExCommsClient
	threadCLI    threading.ThreadsClient
	authCLI      auth.AuthClient
}

func newCleanupSpamAccountsCmd(cnf *config) (command, error) {
	excommsCLI, err := cnf.exCommsClient()
	if err != nil {
		return nil, err
	}

	directoryCLI, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}

	threadCLI, err := cnf.threadingClient()
	if err != nil {
		return nil, err
	}

	authCLI, err := cnf.authClient()
	if err != nil {
		return nil, err
	}

	return &cleanupSpamAccountsCmd{
		cnf:          cnf,
		directoryCLI: directoryCLI,
		threadCLI:    threadCLI,
		excommsCLI:   excommsCLI,
		authCLI:      authCLI,
	}, nil
}

func (c *cleanupSpamAccountsCmd) run(args []string) error {
	fs := flag.NewFlagSet("cleanupspamaccounts", flag.ExitOnError)
	orgEntityFileName := fs.String("org_entity_filename", "", "name of file containing org entities")

	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	if *orgEntityFileName == "" {
		return errors.New("Fileaname for file containing org entities required")
	}

	file, err := os.Open(*orgEntityFileName)
	if err != nil {
		return errors.Trace(err)
	}

	var orgIDs []string
	r := csv.NewReader(file)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		orgIDs = append(orgIDs, row[0])
	}

	for _, orgID := range orgIDs {

		glog := golog.Context("org_id", orgID)

		glog.Infof("Attempting to clean up")
		orgEntity, err := directory.SingleEntity(context.Background(), c.directoryCLI, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: orgID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 1,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
					directory.EntityInformation_MEMBERS,
					directory.EntityInformation_EXTERNAL_IDS,
				},
			},
			RootTypes:  []directory.EntityType{directory.EntityType_ORGANIZATION},
			ChildTypes: []directory.EntityType{directory.EntityType_INTERNAL},
		})
		if err != nil {
			if errors.Cause(err) == directory.ErrEntityNotFound {
				glog.Errorf("Skipping org")
				continue
			}
			return errors.Trace(err)
		}

		// check to ensure that there is only 1 teammate in the team
		if len(orgEntity.Members) != 1 {
			glog.Errorf("Skipping org as %d teammmates found", len(orgEntity.Members))
			continue
		}

		providerEntity := orgEntity.Members[0]

		// check to ensure that there are no more than 4 conversations in total for the provider
		savedQueriesRes, err := c.threadCLI.SavedQueries(context.Background(), &threading.SavedQueriesRequest{
			EntityID: providerEntity.ID,
		})
		if err != nil {
			return errors.Errorf("Unable to lookup saved queries for %s: %s", providerEntity.ID, err)
		}

		// in the All saved query there should be no more than 4 conversations in total
		var allSavedQuery *threading.SavedQuery
		for _, savedQuery := range savedQueriesRes.SavedQueries {
			if savedQuery.ShortTitle == "All" {
				allSavedQuery = savedQuery
				break
			}
		}

		if allSavedQuery == nil {
			return errors.Errorf("All saved query not found for org %s", orgID)
		}

		if allSavedQuery.Total > 4 {
			glog.Errorf("Skipping org as %d conversations found in All saved query", allSavedQuery.Total)
			continue
		}

		queryThreadsRes, err := c.threadCLI.QueryThreads(context.Background(), &threading.QueryThreadsRequest{
			ViewerEntityID: providerEntity.ID,
			Type:           threading.QUERY_THREADS_TYPE_SAVED,
			QueryType: &threading.QueryThreadsRequest_SavedQueryID{
				SavedQueryID: allSavedQuery.ID,
			},
			Iterator: &threading.Iterator{
				Count:     10,
				Direction: threading.ITERATOR_DIRECTION_FROM_END,
			},
		})
		if err != nil {
			return errors.Errorf("Unable to query threads for provider %s: %s", providerEntity.ID, err)
		}

		if len(queryThreadsRes.Edges) > 4 {
			glog.Errorf("Skipping org as expected no more than 4 total conversations but got %d for provider %s", len(queryThreadsRes.Edges), providerEntity.ID)
			continue
		}

		var patientConversations []*threading.Thread
		var spruceSupportConversation *threading.Thread
		for _, edgeItem := range queryThreadsRes.Edges {
			if edgeItem.Thread.Type == threading.THREAD_TYPE_EXTERNAL {
				patientConversations = append(patientConversations, edgeItem.Thread)
			}
			if edgeItem.Thread.Type == threading.THREAD_TYPE_SUPPORT {
				spruceSupportConversation = edgeItem.Thread
			}
		}

		if len(patientConversations) == 0 {
			glog.Errorf("Skipping org as no patient conversation found for provider %s", providerEntity.ID)
			continue
		}

		for _, patientConversation := range patientConversations {
			// the thread should just have a single message
			if patientConversation.MessageCount != 1 {
				glog.Errorf("Skipping org as expected 1 message in conversation but got %d for provider %s", patientConversation.MessageCount, providerEntity.ID)
				continue
			}
			glog.Infof("Summary: %s", patientConversation.LastMessageSummary)
		}

		// its possible the spruce support conversation did not get created for the blocked account
		if spruceSupportConversation != nil {
			// delete linked support thread
			linkedThreadRes, err := c.threadCLI.LinkedThread(context.Background(), &threading.LinkedThreadRequest{
				ThreadID: spruceSupportConversation.ID,
			})
			if err != nil {
				if grpc.Code(err) == codes.NotFound {
					glog.Warningf("No spruce support conversation found for org so nothing to delete")
				} else {
					return errors.Errorf("Unable to find linked spruce support thread for org %s provider %s: %s", orgID, providerEntity.ID, err)
				}
			} else if linkedThreadRes.Thread != nil {
				if _, err := c.threadCLI.DeleteThread(context.Background(), &threading.DeleteThreadRequest{
					ThreadID:      linkedThreadRes.Thread.ID,
					ActorEntityID: spruceSupportConversation.OrganizationID,
				}); err != nil {
					return errors.Errorf("Unable to delete thread %s for org %s provider %s: %s", linkedThreadRes.Thread.ID, orgID, providerEntity.ID, err)
				}
				glog.Infof("Successfully deleted spruce team support thread")
			}
		}

		// deprovision phone number
		var sprucePhoneNumberContact *directory.Contact
		for _, contact := range orgEntity.Contacts {
			if contact.ContactType == directory.ContactType_PHONE && contact.Provisioned {
				sprucePhoneNumberContact = contact
				break
			}
		}
		if sprucePhoneNumberContact != nil {
			if _, err := c.excommsCLI.DeprovisionPhoneNumber(context.Background(), &excomms.DeprovisionPhoneNumberRequest{
				PhoneNumber: sprucePhoneNumberContact.Value,
			}); err != nil {
				return errors.Errorf("Unable to deprovision phone number for org %s provider %s: %s", orgID, providerEntity.ID, err)
			}

			if _, err := c.directoryCLI.DeleteContacts(context.Background(), &directory.DeleteContactsRequest{
				EntityID:         orgEntity.ID,
				EntityContactIDs: []string{sprucePhoneNumberContact.ID},
			}); err != nil {
				return errors.Errorf("Unable to delete contact %s for org %s", sprucePhoneNumberContact.ID, orgEntity.ID)
			}

		}
		glog.Infof("Successfully deprovisioned phone number")

		// block account
		if _, err := c.authCLI.BlockAccount(context.Background(), &auth.BlockAccountRequest{
			AccountID: providerEntity.ExternalIDs[0],
		}); err != nil {
			return errors.Errorf("Unable to block account %s: %s", providerEntity.ExternalIDs[0], err)
		}
		glog.Infof("Successfully blocked account")
	}
	return nil
}
