package version

// Version components
const (
	AppVer = 8
)

var (
	// Version must be a string because scripts like dist.sh read this file.
	Version = "3.4.0"

	// GitCommit is the current HEAD set using ldflags.
	GitCommit string
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit
	}
}
