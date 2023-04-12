package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// newMetricsCmd creates the command exporting metrics.
func (c *cli) newMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "metrics <path>",
		Short:   "metrics",
		Example: "datadog-exporter metrics ./metrics",
		Args:    cobra.ExactArgs(1),

		Run: func(cmd *cobra.Command, args []string) {
			c.log.Info("metric export started")
			ddc := newDDC()

			c.log.Info("listing metrics")
			metrics, err := ddc.metrics(cmd.Context(), "")
			if err != nil {
				c.log.Fatalw("failed to list metrics", zap.Error(err))
			}

			g, ctx := errgroup.WithContext(cmd.Context())
			g.SetLimit(concurrency)

			for _, name := range metrics {
				name := name
				g.Go(func() error {
					log := c.log.With(zap.String("name", name))
					log.Info("exporting metric")

					json, err := ddc.metricJSON(ctx, name)
					if err != nil {
						return fmt.Errorf("failed to get json for metric: %s: %w", name, err)
					}

					err = ddc.writeJSONToFile(filepath.Join(args[0], name), json)
					if err != nil {
						return fmt.Errorf("failed to write json to file for metric: %s: %w", name, err)
					}

					return nil
				})
			}

			err = g.Wait()
			if err != nil {
				c.log.Fatalw("failed to export metrics", zap.Error(err))
			}

			c.log.Info("metric export completed")
		},
	}

	return cmd
}

// newMetricsAnalysisCmd creates the command exporting a metric analysis.
func (c *cli) newMetricsAnalysisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "metrics-analysis <search> <path>",
		Short:   "metrics analysis",
		Example: "datadog-exporter metrics-analysis 'system.cpu.' .",
		Args:    cobra.ExactArgs(2),

		Run: func(cmd *cobra.Command, args []string) {
			c.log.Info("metric analysis export started")
			ddc := newDDC()

			json, err := ddc.metricAnalysisJSON(cmd.Context(), args[0])
			if err != nil {
				c.log.Fatalw("failed to run analysis", zap.Error(err))
			}

			err = ddc.writeJSONToFile(filepath.Join(args[1], "analysis"), json)
			if err != nil {
				c.log.Fatalw("failed to write metric json to file", zap.Error(err))
			}
		},
	}

	return cmd
}
