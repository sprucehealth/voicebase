package cmd

import "github.com/sprucehealth/backend/cmd/cli/deployadmin/internal/config"

type Command interface {
	Run(args []string) error
}

type CommandNew func(*config.Config) (Command, error)
