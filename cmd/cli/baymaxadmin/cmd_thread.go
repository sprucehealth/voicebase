package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type threadCmd struct {
	cnf          *config
	threadingCli threading.ThreadsClient
}

func newThreadCmd(cnf *config) (command, error) {
	threadingCli, err := cnf.threadingClient()
	if err != nil {
		return nil, err
	}
	return &threadCmd{
		cnf:          cnf,
		threadingCli: threadingCli,
	}, nil
}

func (c *threadCmd) run(args []string) error {
	fs := flag.NewFlagSet("thread", flag.ExitOnError)
	threadID := fs.String("thread_id", "", "ID of a thread")
	viewerEntityID := fs.String("viewer_entity_id", "", "Optional viewer entity ID")
	members := fs.Bool("members", false, "Lookup members for the thread")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *threadID == "" {
		*threadID = prompt(scn, "Thread ID: ")
	}
	if *threadID == "" {
		return errors.New("Thread ID is required")
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	res, err := c.threadingCli.Thread(ctx, &threading.ThreadRequest{
		ThreadID:       *threadID,
		ViewerEntityID: *viewerEntityID,
	})
	if grpc.Code(err) == codes.NotFound {
		return errors.New("Thread not found")
	} else if err != nil {
		return fmt.Errorf("Failed to lookup thread: %s", err)
	}

	displayThread(res.Thread)

	if res.Thread.Type == threading.THREAD_TYPE_SUPPORT {
		res, err := c.threadingCli.LinkedThread(ctx, &threading.LinkedThreadRequest{
			ThreadID: res.Thread.ID,
		})
		if err == nil {
			fmt.Printf("\nLinked thread:\n")
			displayThread(res.Thread)
		} else if err != nil && grpc.Code(err) != codes.NotFound {
			return err
		}
	}

	if *members {
		res, err := c.threadingCli.ThreadMembers(ctx, &threading.ThreadMembersRequest{ThreadID: *threadID})
		if err != nil {
			return fmt.Errorf("Failed to lookup thread members: %s", err)
		}
		if len(res.Members) != 0 {
			fmt.Printf("\nMembers:\n")
			for _, m := range res.Members {
				fmt.Printf("\t%s\n", m.EntityID)
			}
		}
		if len(res.FollowerEntityIDs) != 0 {
			fmt.Printf("\nFollowers:\n")
			for _, id := range res.FollowerEntityIDs {
				fmt.Printf("\t%s\n", id)
			}
		}
	}

	return nil
}

func displayThread(t *threading.Thread) {
	fmt.Printf("Thread %s (type %s) (unread %t) (unreadReference %t)\n", t.ID, t.Type, t.Unread, t.UnreadReference)
	fmt.Printf("    Organization ID: %s\n", t.OrganizationID)
	fmt.Printf("    Primary Entity ID: %s\n", t.PrimaryEntityID)
	fmt.Printf("    Last Message Timestamp: %s\n", time.Unix(int64(t.LastMessageTimestamp), 0))
	fmt.Printf("    Created Timestamp: %s\n", time.Unix(int64(t.CreatedTimestamp), 0))
	fmt.Printf("    Message Count: %d\n", t.MessageCount)
	if len(t.LastPrimaryEntityEndpoints) != 0 {
		fmt.Printf("    Last Primary Entity Endpoints:\n")
		w := tabwriter.NewWriter(os.Stdout, 4, 8, 4, ' ', 0)
		fmt.Fprintf(w, "        Channel\tID\n")
		for _, e := range t.LastPrimaryEntityEndpoints {
			fmt.Fprintf(w, "        %s\t%s\n", e.Channel, e.ID)
		}
		if err := w.Flush(); err != nil {
			golog.Fatalf(err.Error())
		}
	}
}
