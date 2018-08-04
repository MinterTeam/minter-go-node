package version

// Version components
const (
	Maj = "0"
	Min = "1"
	Fix = "8"
)

var (
	// Must be a string because scripts like dist.sh read this file.
	Version = "0.1.8"

	// GitCommit is the current HEAD set using ldflags.
	GitCommit string
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit
	}
}
