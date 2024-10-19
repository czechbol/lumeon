package app

// Version and GitCommit values overwritten by `-ldflags` in `go build`.
var (
	serviceName = "lumeon"
	version     = "dev"
	gitCommit   = "dev"
	buildDate   = "1970-01-01T00:00:00Z"
)
