package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
)

const fileMode = 0600

type ddc struct {
	dashAPI *datadogV1.DashboardsApi
}

func newDDC() *ddc {
	dd := datadog.NewAPIClient(datadog.NewConfiguration())

	return &ddc{dashAPI: datadogV1.NewDashboardsApi(dd)}
}

// Dashboards returns a list of all dashboard IDs in the account.
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

// DashboardJSON returns the JSON definition for a given dashboard ID.
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

func (d *ddc) writeJSONToFile(name string, json []byte) error {
	err := os.WriteFile(name+".json", json, fileMode)
	if err != nil {
		return fmt.Errorf("failed to write file for filename: %s: %w", name, err)
	}

	return nil
}
