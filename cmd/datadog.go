package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

const fileMode = 0600

// ddc holds the datadog client and other info for all out export operations.
type ddc struct {
	dashAPI     *datadogV1.DashboardsApi
	monitorAPI  *datadogV1.MonitorsApi
	metricAPI   *datadogV1.MetricsApi
	metricV2API *datadogV2.MetricsApi
}

// newDDC creates a new datadog client.
func newDDC() *ddc {
	dd := datadog.NewAPIClient(datadog.NewConfiguration())

	return &ddc{
		dashAPI:     datadogV1.NewDashboardsApi(dd),
		monitorAPI:  datadogV1.NewMonitorsApi(dd),
		metricAPI:   datadogV1.NewMetricsApi(dd),
		metricV2API: datadogV2.NewMetricsApi(dd),
	}
}

// dashboards returns a list of all dashboard IDs in the account.
func (d *ddc) dashboards(ctx context.Context) ([]string, error) {
	ctx = datadog.NewDefaultContext(ctx)

	list, _, err := d.dashAPI.ListDashboards(ctx, *datadogV1.NewListDashboardsOptionalParameters())
	if err != nil {
		return nil, fmt.Errorf("failed to list dashbaords: %w", err)
	}

	dbs := []string{}
	for _, dash := range list.Dashboards {
		dbs = append(dbs, *dash.Id)
	}

	return dbs, nil
}

// monitors returns a list of all monitor IDs in the account.
func (d *ddc) monitors(ctx context.Context) ([]int64, error) {
	ctx = datadog.NewDefaultContext(ctx)

	list, _, err := d.monitorAPI.ListMonitors(ctx, *datadogV1.NewListMonitorsOptionalParameters())
	if err != nil {
		return nil, fmt.Errorf("failed to list monitors: %w", err)
	}

	mns := []int64{}
	for _, monitor := range list {
		mns = append(mns, *monitor.Id)
	}

	return mns, nil
}

// metrics returns a list of all metrics IDs in the account matching
// the search filter. An empty search will return all metrics.
func (d *ddc) metrics(ctx context.Context, search string) ([]string, error) {
	ctx = datadog.NewDefaultContext(ctx)

	// ListMetrics is deprecated, using ListActiveMetrics with a 24h lookback
	from := time.Now().Add(-24 * time.Hour).Unix()
	list, _, err := d.metricAPI.ListActiveMetrics(ctx, from, *datadogV1.NewListActiveMetricsOptionalParameters())
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics: %w", err)
	}

	if search == "" {
		return list.Metrics, nil
	}

	var out []string
	for _, m := range list.Metrics {
		if strings.Contains(m, search) {
			out = append(out, m)
		}
	}

	return out, nil
}

// metricTags return a map of tag keys to a list of tag values for a given metric name.
func (d *ddc) metricTags(ctx context.Context, name string) (map[string][]string, error) {
	list, _, err := d.metricV2API.ListTagsByMetricName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric tags with name: %s: %w", name, err)
	}

	out := map[string][]string{}
	for _, tag := range list.Data.Attributes.Tags {
		key, value, found := strings.Cut(tag, ":")

		if !found {
			out[key] = []string{}

			continue
		}

		_, ok := out[key]
		if !ok {
			out[key] = []string{}
		}

		out[key] = append(out[key], value)
	}

	return out, nil
}

// metricAnalysis gets all metrics for the given search filter and outputs a list of
// all tags used by those metrics sorted by the cardinality of those tags.
func (d *ddc) metricAnalysis(ctx context.Context, search string) ([][2]string, error) {
	ctx = datadog.NewDefaultContext(ctx)

	metrics, err := d.metrics(ctx, search)
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics: %w", err)
	}

	// create a map of tags to their unique values for all metrics
	allTags := map[string]map[string]bool{}

	for _, metric := range metrics {
		tags, err := d.metricTags(ctx, metric)
		if err != nil {
			return nil, fmt.Errorf("failed to list tags for metric: %s, %w", metric, err)
		}

		for key, values := range tags {
			_, ok := allTags[key]
			if !ok {
				allTags[key] = map[string]bool{}
			}

			for _, value := range values {
				allTags[key][value] = true
			}
		}
	}

	analysis := [][2]string{}
	for key, values := range allTags {
		analysis = append(analysis, [2]string{key, strconv.Itoa(len(values))})
	}

	sort.Slice(analysis, func(i, j int) bool {
		one, _ := strconv.Atoi(analysis[i][1]) //nolint:errcheck // we know that the string is a valid number so we can ignore conversion errors
		two, _ := strconv.Atoi(analysis[j][1]) //nolint:errcheck // we know that the string is a valid number so we can ignore conversion errors

		return one > two
	})

	return analysis, nil
}

// dashboardJSON returns the JSON definition for a given dashboard ID.
func (d *ddc) dashboardJSON(ctx context.Context, id string) ([]byte, error) {
	ctx = datadog.NewDefaultContext(ctx)

	_, resp, err := d.dashAPI.GetDashboard(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard with id: %s: %w", id, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for id: %s: %w", id, err)
	}

	var dash bytes.Buffer
	err = json.Indent(&dash, body, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to indent json for id: %s: %w", id, err)
	}

	return dash.Bytes(), nil
}

// monitorJSON returns the JSON definition for a given monitor ID.
func (d *ddc) monitorJSON(ctx context.Context, id int64) ([]byte, error) {
	ctx = datadog.NewDefaultContext(ctx)

	_, resp, err := d.monitorAPI.GetMonitor(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard with id: %d: %w", id, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for id: %d: %w", id, err)
	}

	var dash bytes.Buffer
	err = json.Indent(&dash, body, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to indent json for id: %d: %w", id, err)
	}

	return dash.Bytes(), nil
}

// metricJSON returns the JSON definition for a given metric name.
func (d *ddc) metricJSON(ctx context.Context, name string) ([]byte, error) {
	ctx = datadog.NewDefaultContext(ctx)

	_, resp, err := d.metricAPI.GetMetricMetadata(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric with name: %s: %w", name, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for name: %s: %w", name, err)
	}

	respMap := map[string]any{}

	err = json.Unmarshal(body, &respMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json for name: %s: %w", name, err)
	}

	tags, err := d.metricTags(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric tags for metric with name: %s: %w", name, err)
	}

	respMap["tags"] = tags

	metric, err := json.MarshalIndent(respMap, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal indent json for name: %s: %w", name, err)
	}

	return metric, nil
}

// metricAnalysisJSON returns the JSON definition for a given metric analysis.
func (d *ddc) metricAnalysisJSON(ctx context.Context, search string) ([]byte, error) {
	ctx = datadog.NewDefaultContext(ctx)

	analysis, err := d.metricAnalysis(ctx, search)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric analysis for search: %s: %w", search, err)
	}

	out, err := json.MarshalIndent(map[string]any{"analysis": analysis}, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metric analysis indent json for search: %s: %w", search, err)
	}

	return out, nil
}

// writeJSONToFile writes json bytes to a local file for archiving.
func (d *ddc) writeJSONToFile(name string, json []byte) error {
	err := os.WriteFile(name+".json", json, fileMode)
	if err != nil {
		return fmt.Errorf("failed to write file for filename: %s: %w", name, err)
	}

	return nil
}
