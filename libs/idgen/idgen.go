// Package idgen provides 64-bit unsigned globally unique ID generation. It does
// so by combining the current time in milliseconds, datacenter ID, machine ID,
// and a sequence number. The datacenter+machine ID must be globally unique.
// Checks are made for time moving backwards and the sequence number wrapping.
// A custom epoch is used for the time to give more headroom. The IDs are locally
// orderable and globally K-orderable in time (unordered within a millisecond but
// strong ordering beyond a millisecond).
//
// - A maximum of 4,096 IDs can be generated per millisecond per worker.
// - IDs will roll over over in the year 2153
package idgen

import (
	"errors"
	"log"
	"net"
	"runtime"
	"sync"
	"time"
)

var (
	// ErrTimeRanBackwards signifies that an ID could not be generated
	// because the system time ran backwards.
	ErrTimeRanBackwards = errors.New("analytics: time went backwards")
)

const (
	// |---------42---------||-----5----||---5--||---12---|
	//  time in milliseconds  datacenter  worker  sequence
	sequenceBits     = 12 // 4,096 IDs per millisecond max per worker
	workerIDBits     = 5
	workerShift      = sequenceBits
	workerMask       = 1<<workerIDBits - 1
	datacenterIDBits = 5
	datacenterShift  = workerShift + workerIDBits
	datacenterMask   = 1<<datacenterIDBits - 1
	timeShift        = datacenterShift + datacenterIDBits
	timeBits         = 63 - timeShift
	maxSequence      = 1<<sequenceBits - 1
	epoch            = 1401316885264
)

var (
	datacenterID uint64
	workerID     uint64
	seq          uint64
	lastTime     uint64
	mu           sync.Mutex
)

func init() {
	// FIXME: This uses the least significant byte of the IP address to generate a unique
	// ID. This works for now since all systems that generate IDs run in the private VPC
	// subnet which has a limited range of IPs. However, this will break in the future
	// when more systems are needed than the current IP range allows.
	iface, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Failed to get network interfaces: %s", err.Error())
	}
	for _, eth := range iface {
		if eth.Name == "eth0" || eth.Name == "en0" {
			addr, err := eth.Addrs()
			if err != nil {
				log.Fatalf("Failed to get addresses for %s: %s", eth.Name, err.Error())
			}
			for _, a := range addr {
				if ip, ok := a.(*net.IPNet); ok {
					if ip4 := ip.IP.To4(); ip4 != nil {
						datacenterID = uint64((ip4[len(ip4)-1] >> workerIDBits) & datacenterMask)
						workerID = uint64(ip4[len(ip4)-1] & workerMask)
					}
				}
			}
		}
	}
}

func now() uint64 {
	return uint64(time.Now().UnixNano() / 1e6) // ms
}

// NewID returns a 64-bit unsigned globally unique ID.
func NewID() (uint64, error) {
	mu.Lock()

	t := now()
	if t < lastTime {
		mu.Unlock()
		return 0, ErrTimeRanBackwards
	}
	if lastTime != t {
		seq = 0
		lastTime = t
	} else {
		seq++
		if seq > maxSequence {
			// Spin waiting for a millisecond to tick. Using Gosched feels better than Sleep
			// for such a short time interval but no idea if it really is.
			for t == lastTime {
				runtime.Gosched()
				t = now()
			}
			seq = 0
		}
	}
	id := (t-epoch)<<timeShift | datacenterID<<datacenterShift | workerID<<workerShift | seq

	mu.Unlock()

	return id, nil
}
