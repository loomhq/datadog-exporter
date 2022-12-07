package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// newDashboardsCmd creates the command exporting dashboards.
func (c *cli) newDashboardsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dashboards <path>",
		Short:   "dashboards",
		Example: "datadog-exporter dashboards ./dashboards",
		Args:    cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			c.log.Info("dashboard export started")
			ddc := newDDC()

			c.log.Info("listing dashboards")
			dashboards, err := ddc.dashboards(cmd.Context())
			if err != nil {
				c.log.Fatalw("failed to list dashboards", zap.Error(err))
			}

			for _, id := range dashboards {
				log := c.log.With(zap.String("id", id))
				log.Info("exporting dashboard")

				json, err := ddc.dashboardJSON(cmd.Context(), id)
				if err != nil {
					c.log.Fatalw("failed to get dashboard json", zap.Error(err))
				}

				err = ddc.writeJSONToFile(filepath.Join(args[0], id), json)
				if err != nil {
					c.log.Fatalw("failed to write dashboard json to file", zap.Error(err))
				}
			}

			c.log.Info("dashboard export completed")
		},
	}

	return cmd
}
