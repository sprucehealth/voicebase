package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sprucehealth/backend/third_party/github.com/armon/consul-api"
)

type consulTTLCheck struct {
	cli   *consulapi.Client
	id    string
	quitC chan chan bool
}

func startConsulCheck(client *consulapi.Client, id string) (*consulTTLCheck, error) {
	// The check may have been left registered from a previous run that crashes in the rare
	// case that this process happens to get the same pid. In this case deregister the old
	// check which will cause the old session to be invalidated. If we don't deregister then
	// the old session could remain valid if we started to use the same check. Unused checks will
	// be left around if the process crashes, but this should hopefully be rare and can be cleaned
	// up by listing check IDs and seeing if the pid is valid.
	for retries := 0; ; retries++ {
		if err := client.Agent().CheckRegister(&consulapi.AgentCheckRegistration{
			ID:   id,
			Name: consulCheckName,
			AgentServiceCheck: consulapi.AgentServiceCheck{
				TTL: consulCheckTTL,
			},
		}); err != nil {
			if !strings.Contains(err.Error(), "already registered") {
				return nil, fmt.Errorf("failed to register consul check: %s", err.Error())
			}
			if err := client.Agent().CheckDeregister(id); err != nil {
				return nil, fmt.Errorf("failed to deregister old consul check: %s", err.Error())
			}
		} else {
			break
		}
	}

	c := &consulTTLCheck{
		cli:   client,
		id:    id,
		quitC: make(chan chan bool),
	}
	c.start()

	return c, nil
}

func (c *consulTTLCheck) start() {
	go func() {
		t := time.NewTicker(time.Second * 30)
		defer func() {
			t.Stop()
		}()

		for {
			select {
			case ch := <-c.quitC:
				if err := c.cli.Agent().CheckDeregister(c.id); err != nil {
					log.Printf("failed to deregister consul check: %s", err.Error())
				}
				ch <- true
				return
			case tm := <-t.C:
				if err := c.cli.Agent().PassTTL(c.id, tm.String()); err != nil {
					log.Printf("Failed to update check TTL: %s", err.Error())
				}
			}
		}
	}()
}

func (c *consulTTLCheck) stop() error {
	ch := make(chan bool)
	c.quitC <- ch
	select {
	case <-ch:
	case <-time.After(time.Second * 5):
	}
	return nil
}
