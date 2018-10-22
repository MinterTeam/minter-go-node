package version

// Version components
const (
	Maj = "0"
	Min = "5"
	Fix = "1"
)

var (
	// Must be a string because scripts like dist.sh read this file.
	Version = "0.5.1"

	// GitCommit is the current HEAD set using ldflags.
	GitCommit string
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit
	}
}
