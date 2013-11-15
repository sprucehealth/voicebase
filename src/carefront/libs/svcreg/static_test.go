package svcreg

import (
	"testing"
)

func TestStaticRegistration(t *testing.T) {
	reg := &StaticRegistry{}
	TestRegistry(t, reg)
}
