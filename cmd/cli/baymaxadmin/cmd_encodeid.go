package main

import (
	"errors"
	"flag"
	"fmt"
	"strconv"

	"github.com/sprucehealth/backend/libs/model"
)

type encodeIDCmd struct {
}

func newEncodeIDCmd(cnf *config) (command, error) {
	return &encodeIDCmd{}, nil
}

func (c *encodeIDCmd) run(args []string) error {
	fs := flag.NewFlagSet("encodeid", flag.ExitOnError)
	prefix := fs.String("prefix", "", "Optional ID prefix")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	if len(args) == 0 {
		return errors.New("ID required")
	}
	id, err := strconv.ParseUint(args[0], 0, 64)
	if err != nil {
		return fmt.Errorf("Invalid ID: %s", err)
	}
	m := model.ObjectID{
		Prefix:  *prefix,
		Val:     id,
		IsValid: true,
	}
	fmt.Println(m.String())
	return nil
}
