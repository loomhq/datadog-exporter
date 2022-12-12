package cmd

import (
	"github.com/spf13/cobra"
)

// newRootCmd creates our base cobra command to add all subcommands to.
func (c *cli) newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "datadog-exporter",
		Short: "datadog-exporter is a tool for exporting and backing up datadog resources",

		// prevents docs from adding promotional message footer
		DisableAutoGenTag: true,
	}

	cmd.AddCommand(
		c.newVersionCmd(),
		c.newDashboardsCmd(),
		c.newMonitorsCmd(),
	)

	return cmd
}
