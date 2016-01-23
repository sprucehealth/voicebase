package server

import (
	"github.com/sprucehealth/backend/libs/idgen"
)

type idGenerator interface {
	NewID() (uint64, error)
}

type idg struct{}

func NewIDGenerator() idGenerator {
	return &idg{}
}

func (*idg) NewID() (uint64, error) {
	return idgen.NewID()
}

type mockIDGen struct{}

func newMockIDGen() idGenerator {
	return &mockIDGen{}
}

func (*mockIDGen) NewID() (uint64, error) {
	return uint64(1), nil
}