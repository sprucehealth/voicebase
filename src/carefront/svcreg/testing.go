package svcreg

import (
	"testing"
	"time"
)

func TestRegistry(t *testing.T, reg Registry) {
	id := ServiceId{"prod", "service"}
	member := Member{
		Status: StatusAlive,
		Endpoint: Endpoint{
			Host: "127.0.0.1",
			Port: 1234,
		},
	}

	preWatch := make(chan []ServiceUpdate, 1)
	if err := reg.WatchService(id, preWatch); err != nil {
		t.Fatal(err)
	}

	select {
	case _ = <-preWatch:
		t.Fatalf("Received watch event before any service registrations")
	default:
	}

	svc, err := reg.Register(id, member)
	if err != nil {
		t.Fatal(err)
	}
	if sreg, ok := reg.(*StaticRegistry); ok {
		if members := sreg.Services[id]; members == nil {
			t.Fatalf("Service didn't register: member is nil")
		} else if _, ok := members[member.Endpoint]; !ok {
			t.Fatalf("Service didn't register: member not found")
		}
	}

	select {
	case up := <-preWatch:
		if len(up) != 1 {
			t.Fatalf("Expected 1 update got %d", len(up))
		}
		if up[0].Type != Add {
			t.Fatalf("Expected Add got %d", up[0].Type)
		}
		if up[0].Member.Endpoint != member.Endpoint {
			t.Fatalf("Expected %v got %v", member.Endpoint, up[0].Member.Endpoint)
		}
	case <-time.After(time.Second * 4):
		t.Fatalf("Watch created before a register didn't receive the registration")
	}

	select {
	case _ = <-preWatch:
		t.Fatalf("Received extra watch event")
	default:
	}

	postWatch := make(chan []ServiceUpdate, 1)
	if err := reg.WatchService(id, postWatch); err != nil {
		t.Fatal(err)
	}

	select {
	case up := <-postWatch:
		if len(up) != 1 {
			t.Fatalf("Expected 1 update got %d", len(up))
		}
		if up[0].Type != Add {
			t.Fatalf("Expected Add got %d", up[0].Type)
		}
		if up[0].Member.Endpoint != member.Endpoint {
			t.Fatalf("Expected %v got %v", member.Endpoint, up[0].Member.Endpoint)
		}
	case <-time.After(time.Second * 4):
		t.Fatalf("Watch ccreated after a register didn't receive the registration")
	}

	select {
	case _ = <-postWatch:
		t.Fatalf("Received extra watch event")
	default:
	}

	if err := svc.Unregister(); err != nil {
		t.Fatal(err)
	}
	if sreg, ok := reg.(*StaticRegistry); ok {
		members := sreg.Services[id]
		if _, ok := members[member.Endpoint]; ok {
			t.Fatalf("Service didn't unregister")
		}
	}

	select {
	case up := <-preWatch:
		if len(up) != 1 {
			t.Fatalf("Expected 1 update got %d", len(up))
		}
		if up[0].Type != Remove {
			t.Fatalf("Expected Remove got %d", up[0].Type)
		}
		if up[0].Member.Endpoint != member.Endpoint {
			t.Fatalf("Expected %v got %v", member.Endpoint, up[0].Member.Endpoint)
		}
	case <-time.After(time.Second * 4):
		t.Fatalf("Watch channel didn't receive unregister")
	}

	select {
	case _ = <-preWatch:
		t.Fatalf("Received extra watch event after unregister")
	default:
	}
}
