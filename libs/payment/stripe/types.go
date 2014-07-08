package stripe

import (
	"strconv"
	"time"
)

type Timestamp struct {
	time.Time
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	s := string(b)

	if s == "null" {
		*t = Timestamp{}
		return nil
	}

	ts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}

	*t = Timestamp{time.Unix(ts, 0)}
	return nil
}
