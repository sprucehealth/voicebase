package zksvcreg

import (
	"encoding/json"
	"log"
	"time"

	"github.com/sprucehealth/backend/libs/svcreg"

	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-zookeeper/zk"
)

type registeredService struct {
	reg      *registry
	id       svcreg.ServiceId
	member   svcreg.Member
	basePath string
	path     string
	stopCh   chan bool
}

func (rs *registeredService) start() error {
	rs.stopCh = make(chan bool)
	data, err := json.MarshalIndent(rs.member, "", "    ")
	if err != nil {
		return err
	}
	go func() {
		lastPath := ""
		for {
			select {
			case <-rs.stopCh:
				return
			default:
			}

			if lastPath != "" {
				if err := rs.reg.zkConn.Delete(lastPath, -1); err != nil && err != zk.ErrNoNode {
					log.Printf("zksvcreg/registered: failed to delete old path: %s", lastPath)
				}
				lastPath = ""
			}

			path, err := rs.reg.zkConn.CreateProtectedEphemeralSequential(rs.basePath+"/member_", data, zk.WorldACL(zk.PermAll))
			if err != nil {
				log.Printf("zksvcreg/registered: failed to create member node: %s", err.Error())
				time.Sleep(time.Second * 5)
				continue
			}
			lastPath = path

			_, stat, ch, err := rs.reg.zkConn.GetW(path)
			if err != nil {
				log.Printf("zksvcreg/registered: error while monitoring registered service: %s", err.Error())
				continue
			}

			select {
			case <-rs.stopCh:
				if err := rs.reg.zkConn.Delete(path, stat.Version); err != nil {
					log.Printf("zksvcreg/registered: failed to delete path during unregister of '%s': %+v", path, err)
				}
				return
			case ev := <-ch:
				if ev.Err != nil {
					log.Printf("zksvcreg/registered: error while monitoring registered service: %s", ev.Err.Error())
				}
			}
		}
	}()

	return nil
}

func (rs *registeredService) stop() {
	close(rs.stopCh)
}

func (rs *registeredService) Unregister() error {
	rs.stop()
	rs.reg.removeRegistered(rs)
	return nil
}
