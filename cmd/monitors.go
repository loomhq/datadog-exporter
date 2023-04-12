package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// newMonitorsCmd creates the command exporting monitors.
func (c *cli) newMonitorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "monitors <path>",
		Short:   "monitors",
		Example: "datadog-exporter monitors ./monitors",
		Args:    cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			c.log.Info("monitor export started")
			ddc := newDDC()

			c.log.Info("listing monitors")
			monitors, err := ddc.monitors(cmd.Context())
			if err != nil {
				c.log.Fatalw("failed to list monitors", zap.Error(err))
			}

			g, ctx := errgroup.WithContext(cmd.Context())
			g.SetLimit(concurrency)

			for _, id := range monitors {
				id := id
				g.Go(func() error {
					log := c.log.With(zap.Int64("id", id))
					log.Info("exporting monitor")

					json, err := ddc.monitorJSON(ctx, id)
					if err != nil {
						return fmt.Errorf("failed to get json for monitor: %d: %w", id, err)
					}

					err = ddc.writeJSONToFile(filepath.Join(args[0], strconv.FormatInt(id, 10)), json)
					if err != nil {
						return fmt.Errorf("failed to write json for monitor: %d: %w", id, err)
					}

					return nil
				})
			}

			err = g.Wait()
			if err != nil {
				c.log.Fatalw("failed to export monitors", zap.Error(err))
			}

			c.log.Info("monitor export completed")
		},
	}

	return cmd
}
