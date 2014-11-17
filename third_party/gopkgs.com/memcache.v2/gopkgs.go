package memcache

import (
	"fmt"
	"reflect"
)

// gopkgs.go: v1

// NOTE: This file is autogenerated by gopkgs.com.
const (
	goPkgsSrcPath = "github.com/rainycape/memcache"
	goPkgsName    = "memcache"
	goPkgsErrFmt  = "invalid import path %s - please use gopkgs.com/%s.v2 or see http://gopkgs.com/%s"
)

type goPkgsCheck struct{}

func init() {
	typ := reflect.TypeOf(goPkgsCheck{})
	if typ.PkgPath() == goPkgsSrcPath {
		panic(fmt.Errorf(goPkgsErrFmt, typ.PkgPath(), goPkgsName, goPkgsName))
	}
}
