package helm

import (
	"fmt"

	"helm.sh/helm/v4/pkg/action"
	chartpkg "helm.sh/helm/v4/pkg/chart"
	"helm.sh/helm/v4/pkg/chart/loader"
	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/cli"
)

// LoadResult is the result of locating and loading an installable chart.
type LoadResult struct {
	Chart    *chartv2.Chart
	Warnings []string
}

// LoadInstallable locates, loads, and validates an installable Helm chart.
func LoadInstallable(envSettings *cli.EnvSettings, chartRef, version string) (*LoadResult, error) {
	chartPathOptions := action.ChartPathOptions{Version: version}
	path, err := chartPathOptions.LocateChart(chartRef, envSettings)
	if err != nil {
		return nil, fmt.Errorf("locate chart: %w", err)
	}

	loadedChart, err := loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}

	chart, ok := loadedChart.(*chartv2.Chart)
	if !ok {
		return nil, fmt.Errorf("expected *chart.Chart, got %T", loadedChart)
	}

	if chart.Metadata == nil {
		return nil, fmt.Errorf("chart metadata is missing")
	}

	if chart.Metadata.Type != "application" && chart.Metadata.Type != "" {
		return nil, fmt.Errorf("%s charts are not installable", chart.Metadata.Type)
	}

	if deps := chart.Metadata.Dependencies; deps != nil {
		reqs := make([]chartpkg.Dependency, len(deps))
		for i, dep := range deps {
			reqs[i] = dep
		}

		if err := action.CheckDependencies(chart, reqs); err != nil {
			return nil, fmt.Errorf("check chart dependencies: %w", err)
		}
	}

	result := &LoadResult{Chart: chart}
	if chart.Metadata.Deprecated {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Helm chart %s is deprecated", chart.Metadata.Name))
	}

	return result, nil
}
