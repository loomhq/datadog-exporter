package cmd

import (
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
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

			for _, id := range monitors {
				log := c.log.With(zap.Int64("id", id))
				log.Info("exporting monitor")

				json, err := ddc.monitorJSON(cmd.Context(), id)
				if err != nil {
					c.log.Fatalw("failed to get monitor json", zap.Error(err))
				}

				err = ddc.writeJSONToFile(filepath.Join(args[0], strconv.FormatInt(id, 10)), json)
				if err != nil {
					c.log.Fatalw("failed to write monitor json to file", zap.Error(err))
				}
			}

			c.log.Info("monitor export completed")
		},
	}

	return cmd
}
