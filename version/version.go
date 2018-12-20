package version

// Version components
const (
	Maj = "0"
	Min = "9"
	Fix = "0"

	AppVersion = 1
)

var (
	// Must be a string because scripts like dist.sh read this file.
	Version = "0.9.0"

	// GitCommit is the current HEAD set using ldflags.
	GitCommit string
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit
	}
}
