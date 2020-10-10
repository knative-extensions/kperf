package version

import (
	"github.com/spf13/cobra"
)

var Version string
var BuildDate string
var GitRevision string

// NewVersionCommand implements 'kn version' command
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Prints the kperf version",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Printf("Version:      %s\n", Version)
			cmd.Printf("Build Date:   %s\n", BuildDate)
			cmd.Printf("Git Revision: %s\n", GitRevision)
			return nil
		},
	}
}
