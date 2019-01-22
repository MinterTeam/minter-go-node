package version

// Version components
const (
	Maj = "0"
	Min = "10"
	Fix = "1"

	AppVer = 2
)

var (
	// Must be a string because scripts like dist.sh read this file.
	Version = "0.10.1"

	// GitCommit is the current HEAD set using ldflags.
	GitCommit string
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit
	}
}
