package version

// Version components
const (
	Maj = "0"
	Min = "9"
	Fix = "5"

	AppVer = 1
)

var (
	// Must be a string because scripts like dist.sh read this file.
	Version = "0.9.5"

	// GitCommit is the current HEAD set using ldflags.
	GitCommit string
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit
	}
}
