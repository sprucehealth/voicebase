package svcreg

import (
	"net"
	"os"
	"regexp"
)

var ipv4re = regexp.MustCompile(`^[0-9\.]+$`)

// Helper function to get current machine's ip address (prefers IPv4)
func Addr() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "", err
	}
	addrs, err := net.LookupHost(name)
	if err != nil {
		return "", err
	}
	// Try to find an IPv4 first
	for _, a := range addrs {
		if ipv4re.MatchString(a) {
			return a, nil
		}
	}
	// Fallback to the first address
	return addrs[0], nil
}
