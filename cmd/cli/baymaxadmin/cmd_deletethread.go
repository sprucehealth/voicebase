package main

import (
	"bufio"
	"context"
	"flag"
	"os"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/threading"
)

type deleteThreadCmd struct {
	cnf       *config
	threading threading.ThreadsClient
}

func newDeleteThreadCmd(cnf *config) (command, error) {
	threadCli, err := cnf.threadingClient()
	if err != nil {
		return nil, err
	}

	return &deleteThreadCmd{
		cnf:       cnf,
		threading: threadCli,
	}, nil
}

func (d *deleteThreadCmd) run(args []string) error {
	fs := flag.NewFlagSet("deletethread", flag.ExitOnError)
	threadID := fs.String("thread_id", "", "ID of the thread to delete")
	requestingEntityID := fs.String("requesting_entity_id", "", "id of the entity requesting the delete")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	ctx := context.Background()

	scn := bufio.NewScanner(os.Stdin)

	if *threadID == "" {
		*threadID = prompt(scn, "Thread ID: ")
	}
	if *threadID == "" {
		return errors.New("Thread ID required")
	}

	if *requestingEntityID == "" {
		*requestingEntityID = prompt(scn, "Requesting Entity ID: ")
	}
	if *requestingEntityID == "" {
		return errors.New("Requesting Entity ID required")
	}

	if _, err := d.threading.Thread(ctx, &threading.ThreadRequest{
		ThreadID: *threadID,
	}); err != nil {
		return errors.Trace(err)
	}

	if _, err := d.threading.DeleteThread(ctx, &threading.DeleteThreadRequest{
		ThreadID:      *threadID,
		ActorEntityID: *requestingEntityID,
	}); err != nil {
		return errors.Trace(err)
	}

	return nil
}
