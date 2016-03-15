package main

import (
	"net"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type blockChinaHandler struct {
	h httputil.ContextHandler
}

var chinaIPv4CIDRRanges map[string][]*net.IPNet
var chinaIPv6CIDRRanges []*net.IPNet

func init() {
	chinaIPv4CIDRRanges = make(map[string][]*net.IPNet, len(chinaIPv4CIDRStrings))

	for _, s := range chinaIPv4CIDRStrings {
		_, ipnet, err := net.ParseCIDR(s)
		if err != nil {
			panic(err)
		}
		firstOctet := s[:strings.IndexRune(s, '.')]
		ranges := chinaIPv4CIDRRanges[firstOctet]
		chinaIPv4CIDRRanges[firstOctet] = append(ranges, ipnet)
	}

	chinaIPv6CIDRRanges = make([]*net.IPNet, len(chinaIPv6CIDRStrings))
	for i, s := range chinaIPv6CIDRStrings {
		_, ipnet, err := net.ParseCIDR(s)
		if err != nil {
			panic(err)
		}
		chinaIPv6CIDRRanges[i] = ipnet
	}
}

func newBlockChinaHandler(h httputil.ContextHandler) httputil.ContextHandler {
	return &blockChinaHandler{
		h: h,
	}
}

func (b *blockChinaHandler) ServeHTTP(context context.Context, w http.ResponseWriter, r *http.Request) {
	remoteAddr := remoteAddrFromRequest(r, *flagBehindProxy)

	// if the remoteAddr is from China, block access
	if ipAddressFromChina(remoteAddr) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	b.h.ServeHTTP(context, w, r)
}

func ipAddressFromChina(remoteAddr string) bool {
	ip := net.ParseIP(remoteAddr)

	switch ip4or6(remoteAddr) {
	case ipv4:

		firstOctet := remoteAddr[:strings.IndexRune(remoteAddr, '.')]
		ranges := chinaIPv4CIDRRanges[firstOctet]
		if len(ranges) == 0 {
			return false
		}

		for _, cidr := range ranges {
			if cidr.Contains(ip) {
				return true
			}
		}

	case ipv6:
		for _, cidr := range chinaIPv6CIDRRanges {
			if cidr.Contains(ip) {
				return true
			}
		}
	}

	return false
}

const (
	unknown = -1
	ipv4    = 1
	ipv6    = 2
)

func ip4or6(ip string) int64 {
	for _, r := range ip {
		switch r {
		case ':':
			return ipv6
		case '.':
			return ipv4
		}
	}
	return unknown
}
