package zksvcreg

import (
	"testing"

	"github.com/sprucehealth/backend/libs/svcreg"

	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-zookeeper/zk"
)

func TestReg(t *testing.T) {
	tc, err := zk.StartTestCluster(1)
	if err != nil {
		if err.Error() == "zk: unable to find server jar" {
			t.Skip("Unable to find Zookeeper jar file. Skipping zksvcreg tests.")
		}
		t.Fatal(err)
	}
	defer tc.Stop()
	conn, err := tc.ConnectAll()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	reg, err := NewServiceRegistry(conn, "/test/services")
	if err != nil {
		t.Fatal(err)
	}
	defer reg.Close()

	svcreg.TestRegistry(t, reg)
}
