package svcreg

type MemberStatus string

const (
	StatusAlive        MemberStatus = "ALIVE"
	StatusDead         MemberStatus = "DEAD"
	StatusOutOfService MemberStatus = "OUT_OF_SERVICE"
)

type ServiceId struct {
	Environment string `json:"env"`
	Name        string `json:"name"`
}

type Endpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type Member struct {
	Status              MemberStatus        `json:"status"`
	Endpoint            Endpoint            `json:"endpoint"`
	AdditionalEndpoints map[string]Endpoint `json:"additionalEndpoints"`
}

type UpdateType int

const (
	Add    UpdateType = 0
	Remove UpdateType = iota
)

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
