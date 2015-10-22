package server

import (
	"github.com/spf13/pflag"
)

// SetupCommandLineFlags sets all necessary command line flags and their defaults
func SetupCommandLineFlags(fs *pflag.FlagSet) {
	setupHTTPCommandLineFlags(fs)
}
