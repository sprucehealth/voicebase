package dal

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testsql"
)

const schemaGlob = "./mysql/*.sql"

func TestMedia(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	dal := New(dt.DB)

	med := &Media{
		ID:         "media1",
		MimeType:   "image/jpeg",
		OwnerType:  MediaOwnerTypeEntity,
		OwnerID:    "entity1",
		SizeBytes:  1234,
		DurationNS: 1000,
		Public:     true,
		Name:       "someimage",
	}
	_, err := dal.InsertMedia(med)
	test.OK(t, err)

	m, err := dal.Media("media1")
	test.OK(t, err)
	med.Created = m.Created
	test.Equals(t, med, m)
}
