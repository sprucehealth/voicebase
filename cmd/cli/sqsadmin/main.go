package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/cli/sqsadmin/internal/cmd"
	"github.com/sprucehealth/backend/cmd/cli/sqsadmin/internal/config"
	"github.com/sprucehealth/backend/libs/golog"
)

var commands = map[string]cmd.CommandNew{
	"list_queues": cmd.NewListQueuesCmd,
	"poll_queue":  cmd.NewPollQueueCmd,
}

func main() {
	golog.Default().SetLevel(golog.INFO)
	app := boot.NewApp()
	flag.Parse()

	cmd := flag.Arg(0)

	for name, cfn := range commands {
		if name == cmd {
			c, err := cfn(&config.Config{App: app})
			if err != nil {
				golog.Fatalf(err.Error())
			}
			if err := c.Run(flag.Args()[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "FAILED: %s\n", err)
				os.Exit(2)
			}
			os.Exit(0)
		}
	}

	if cmd != "" {
		fmt.Printf("Unknown command '%s'\n", cmd)
	}

	fmt.Printf("Available commands:\n")
	cmdList := make([]string, 0, len(commands))
	for name := range commands {
		cmdList = append(cmdList, name)
	}
	sort.Strings(cmdList)
	for _, name := range cmdList {
		fmt.Printf("\t%s\n", name)
	}
}
