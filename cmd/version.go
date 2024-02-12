package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newVersionCmd creates a new command that prints the version.
func (c *cli) newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "print the current version",
		Example: "datadog-exporter version",
		Args:    cobra.NoArgs,

		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(c.version)
		},
	}

	return cmd
}
