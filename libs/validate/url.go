package validate

import (
	"net"
	"strings"
)

// RemoteHost returns true iff the provided host is a valid remove domain (not a local IP address).
// Otherwise it returns false the reason.
func RemoteHost(host string) (string, bool) {
	if i := strings.LastIndexByte(host, '.'); i <= 0 {
		return "invalid host", false
	} else if !TLD(host[i+1:]) {
		return "bad TLD", false
	}
	// TODO: support IPv6 - requires changes the local IP checks
	ipa, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return "failed to resolve host", false
	}
	ip := ipa.IP.To4()
	switch {
	case ip.IsMulticast():
		return "multicast disallowed", false
	case ip.IsInterfaceLocalMulticast():
		return "local multicast disallowed", false
	case ip.IsUnspecified():
		return "unspecified ip", false
	case ip.IsLoopback():
		return "loopback disallowed", false
	case ip.IsLinkLocalUnicast():
		return "link local unicast disallowed", false
	case ip.IsLinkLocalMulticast():
		return "link local multicast disallowed", false
	case ip[0] == 10: // 10.0.0.0/8
		return "local ip disallowed", false
	case ip[0] == 172 && (ip[2]&0xf0) == 0x10: // 172.16.0.0/12
		return "local ip disallowed", false
	case ip[0] == 192 && ip[1] == 168: // 192.168.0.0/16
		return "local ip disallowed", false
	}
	return "", true
}
