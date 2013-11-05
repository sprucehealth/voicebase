package svcreg

import (
	"fmt"
)

type ServiceId struct {
	Environment string `json:"env"`
	Name        string `json:"name"`
}

type Endpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (e Endpoint) String() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

type Member struct {
	Endpoint            Endpoint            `json:"endpoint"`
	AdditionalEndpoints map[string]Endpoint `json:"additionalEndpoints"`
}

type UpdateType int

const (
	Add    UpdateType = 0
	Remove UpdateType = iota
)

func (ut UpdateType) String() string {
	switch ut {
	case Add:
		return "Add"
	case Remove:
		return "Remove"
	}
	return "Unknown"
}

type ServiceUpdate struct {
	Type   UpdateType
	Member Member
}

type RegisteredService interface {
	Unregister() error
}

type Registry interface {
	WatchService(ServiceId, chan<- []ServiceUpdate) error
	UnwatchService(ServiceId, chan<- []ServiceUpdate) error
	Register(ServiceId, Member) (RegisteredService, error)
	Close() error
}
