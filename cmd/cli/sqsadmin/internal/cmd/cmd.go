package cmd

import "github.com/sprucehealth/backend/cmd/cli/sqsadmin/internal/config"

type Command interface {
	Run(args []string) error
}

type CommandNew func(*config.Config) (Command, error)
