package main

import (
	"flag"
	"fmt"
	"github.com/spf13/cobra"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/logger"
	"os"
)

var log = logger.New("o-neko")
var rootCmd *cobra.Command

// GitCommit and GitTag are set during compilation
var GitCommit = ""
var GitTag = ""

var commandVersion string
func init() {
	if len(GitTag) == 0 && len(GitCommit) == 0 {
		commandVersion = "dev"
	} else if len(GitTag) == 0 {
		commandVersion = fmt.Sprintf("%s", GitCommit)
	} else {
		commandVersion = fmt.Sprintf("%s (%s)", GitTag, GitCommit)
	}
	rootCmd = &cobra.Command{
		Use:          "o-neko-url-trigger [flags]",
		Short:        "This tool starts O-Neko deployments by its URL when used as a default HTTP backend.",
		Long:         "This tool starts stopped O-Neko deployments by its URL when used as a default HTTP backend in your infrastructure.",
		SilenceUsage: true,
		Version: commandVersion,
	}
	flags := rootCmd.PersistentFlags()
	flags.AddGoFlagSet(flag.CommandLine)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Error(err, "command failed")
		os.Exit(1)
	}
}

func GetVersion() string {
	return commandVersion
}
