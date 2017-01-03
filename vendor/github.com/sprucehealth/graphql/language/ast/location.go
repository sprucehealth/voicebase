package ast

import (
	"github.com/sprucehealth/graphql/language/source"
)

type Location struct {
	Start  int
	End    int
	Source *source.Source
}
