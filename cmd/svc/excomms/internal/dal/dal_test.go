package dal

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testsql"
)

const schemaGlob = "schema/*.sql"

func TestMedia(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	clk := clock.NewManaged(time.Unix(1e9, 0))

	dal := New(dt.DB, clk)

	med := &models.Media{
		ID:   "media1",
		Type: "image/jpeg",
		Name: ptr.String("boo"),
	}
	test.OK(t, dal.StoreMedia([]*models.Media{med}))
	ms, err := dal.LookupMedia([]string{"media1"})
	test.OK(t, err)
	test.Equals(t, 1, len(ms))
	test.Equals(t, med, ms["media1"])
}
