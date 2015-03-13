package financial

import "time"

type Financial interface {
	IncomingItems(from, to time.Time) ([]*IncomingItem, error)
	OutgoingItems(from, to time.Time) ([]*OutgoingItem, error)
}
