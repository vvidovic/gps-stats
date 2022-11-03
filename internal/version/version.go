package version

// Version (shoud be git tag), Platform & Time contains application information
// which can be set during build time (using ldflags -X) and is shown
// when "-v" flag is passed to gps-stats.
var (
	Version   = "development"
	Platform  = "local"
	BuildTime = ""
)
