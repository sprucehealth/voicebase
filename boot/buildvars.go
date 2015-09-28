package boot

import (
	"expvar"
	"runtime"
)

// These variables are set during linking
var (
	GitBranch       string
	GitRevision     string
	BuildTime       string
	BuildNumber     string // CI build number
	MigrationNumber string // The database needs to match this migration number for this build
)

// VersionInfo is a set of build version variables set during linking
var VersionInfo map[string]string

func init() {
	VersionInfo = map[string]string{
		"GitBranch":       GitBranch,
		"GitRevision":     GitRevision,
		"BuildTime":       BuildTime,
		"BuildNumber":     BuildNumber,
		"MigrationNumber": MigrationNumber,
		"GoVersion":       runtime.Version(),
	}

	expvar.Publish("version", expvar.Func(func() interface{} {
		return VersionInfo
	}))
}
