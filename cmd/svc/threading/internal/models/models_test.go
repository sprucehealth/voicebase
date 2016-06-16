package models

import (
	"testing"

	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/svc/threading"
)

func TestThreadID(t *testing.T) {
	t.Parallel()
	var id ThreadID
	id.Prefix = threading.ThreadIDPrefix

	// Empty/invalid state marshaling
	b, err := id.MarshalText()
	test.OK(t, err)
	test.Equals(t, []byte(nil), b)
	test.Equals(t, "", id.String())

	// Valid unmarshaling
	id, err = ParseThreadID("t_00000000002D4")
	test.OK(t, err)
	test.Equals(t, uint64(1234), id.Val)
	test.Equals(t, true, id.IsValid)

	// Valid marshaling
	b, err = id.MarshalText()
	test.OK(t, err)
	test.Equals(t, []byte("t_00000000002D4"), b)
	test.Equals(t, "t_00000000002D4", id.String())
}

func TestThreadIDSort(t *testing.T) {
	t.Parallel()
	id1 := ThreadID{
		model.ObjectID{
			Prefix:  threading.ThreadIDPrefix,
			Val:     2,
			IsValid: true,
		},
	}
	id2 := ThreadID{
		model.ObjectID{
			Prefix:  threading.ThreadIDPrefix,
			Val:     1,
			IsValid: true,
		},
	}
	ids := []ThreadID{id1, id2}
	SortThreadID(ids)
	test.Equals(t, ids[0], id2)
	test.Equals(t, ids[1], id1)
}
