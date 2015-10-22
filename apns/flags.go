package apns

import (
	"github.com/spf13/pflag"
)

// SetupCommandLineFlags sets all necessary command line flags and their defaults
func SetupCommandLineFlags(fs *pflag.FlagSet) {
	setupClientCommandLineFlags(fs)
	setupWorkerCommandLineFlags(fs)
}
