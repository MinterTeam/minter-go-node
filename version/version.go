package version

// Version components
const (
	AppVer = 7
)

var (
	// Version must be a string because scripts like dist.sh read this file.
	Version = "3.0.2"

	// GitCommit is the current HEAD set using ldflags.
	GitCommit string
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit
	}
}
