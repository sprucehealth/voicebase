package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/libs/model"
)

type decodeIDCmd struct {
}

func newDecodeIDCmd(cnf *config) (command, error) {
	return &decodeIDCmd{}, nil
}

func (c *decodeIDCmd) run(args []string) error {
	if len(args) == 0 {
		return errors.New("ID required")
	}
	id, err := decodeID(args[0])
	if err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

func decodeID(encodedID string) (uint64, error) {
	// Remove the prefix, don't actually care what it is
	if i := strings.IndexByte(encodedID, '_'); i >= 0 {
		encodedID = encodedID[i+1:]
	}
	id := model.ObjectID{}
	if err := id.UnmarshalText([]byte(encodedID)); err != nil {
		return 0, fmt.Errorf("Invalid ID: %s", err)
	}
	return id.Val, nil
}
