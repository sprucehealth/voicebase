package main

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"

	"google.golang.org/grpc"

	"flag"
)

var (
	flagThreadingAddr = flag.String("threading_addr", "", "`host:port` of threading service")
	flagDirectoryAddr = flag.String("directory_addr", "", "`host:port` of directory service")
	flagThreadListCSV = flag.String("thread_csv", "", "filename of file containing list of threadIDs that need to have their system title determined")
)

func main() {
	flag.Parse()

	conn, err := grpc.Dial(
		*flagThreadingAddr,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with threading service: %s", err.Error())
	}
	threadingClient := threading.NewThreadsClient(conn)

	conn, err = grpc.Dial(
		*flagDirectoryAddr,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with directory service: %s", err.Error())
	}
	directoryClient := directory.NewDirectoryClient(conn)

	threadIDs, err := getThreadIDs()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	for _, threadID := range threadIDs {
		res, err := threadingClient.Thread(context.Background(), &threading.ThreadRequest{
			ThreadID: threadID,
		})
		if err != nil {
			golog.Errorf("Unable to lookup thread %s: %s", threadID, err.Error())
			continue
		} else if res.Thread == nil {
			golog.Errorf("Expected thread %s to exist but got back null thread", threadID)
			continue
		}
		thread := res.Thread

		var systemTitle string
		switch thread.Type {
		case threading.ThreadType_EXTERNAL, threading.ThreadType_LEGACY_TEAM, threading.ThreadType_SUPPORT:
			if thread.PrimaryEntityID == "" {
				golog.Fatalf("No primaryEntityID for EXTERNAL thread %s", threadID)
			}

			// lookup the primary entity
			res, err := directoryClient.LookupEntities(
				context.Background(),
				&directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: thread.PrimaryEntityID,
					},
				})
			if err != nil {
				golog.Fatalf("Unable to lookup entity information for %s: %s", thread.PrimaryEntityID, err.Error())
			} else if len(res.Entities) != 1 {
				golog.Fatalf("Expected 1 entity for %s but got %d", thread.PrimaryEntityID, len(res.Entities))
			}
			systemTitle = res.Entities[0].Info.DisplayName

		case threading.ThreadType_SETUP:
			systemTitle = "Setup"

		case threading.ThreadType_TEAM:
			// there should be no team thread that needs updating
			golog.Fatalf("TEAM thread %s encountered with empty system title", threadID)

		case threading.ThreadType_UNKNOWN:
			golog.Fatalf("Thread of UNKNOWN type encountered: %s", threadID)
		}

		if _, err := threadingClient.UpdateThread(context.Background(), &threading.UpdateThreadRequest{
			ThreadID:    thread.ID,
			SystemTitle: systemTitle,
		}); err != nil {
			golog.Fatalf("Unable to update thread %s: %s", threadID, err.Error())
		}
		golog.Infof("Successfully updated system title for thread %s", threadID)
	}

}

func getThreadIDs() ([]string, error) {
	csvFile, err := os.Open(*flagThreadListCSV)
	if err != nil {
		return nil, err
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = '\n'

	var threadIDs []string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		threadIDInt64, err := strconv.ParseUint(row[0], 10, 64)
		if err != nil {
			return nil, err
		}

		threadID := model.ObjectID{
			Prefix:  threading.ThreadIDPrefix,
			Val:     threadIDInt64,
			IsValid: true,
		}

		threadIDs = append(threadIDs, threadID.String())
	}

	return threadIDs, nil
}
