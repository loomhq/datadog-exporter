package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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

			g, ctx := errgroup.WithContext(cmd.Context())
			g.SetLimit(concurrency)

			for _, id := range dashboards {
				id := id
				g.Go(func() error {
					log := c.log.With(zap.String("id", id))
					log.Info("exporting dashboard")

					json, err := ddc.dashboardJSON(ctx, id)
					if err != nil {
						return fmt.Errorf("failed to get json for dashboard: %s: %w", id, err)
					}

					err = ddc.writeJSONToFile(filepath.Join(args[0], id), json)
					if err != nil {
						return fmt.Errorf("failed to write json for dashboard: %s: %w", id, err)
					}

					return nil
				})
			}

			err = g.Wait()
			if err != nil {
				c.log.Fatalw("failed to export dashboards", zap.Error(err))
			}

			c.log.Info("dashboard export completed")
		},
	}

	return cmd
}
