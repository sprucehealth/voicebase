package zksvcreg

import (
	"testing"

	"carefront/svcreg"
	"github.com/samuel/go-zookeeper/zk"
)

func TestReg(t *testing.T) {
	tc, err := zk.StartTestCluster(1)
	if err != nil {
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
