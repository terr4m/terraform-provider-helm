package helm

import (
	"path/filepath"
	"testing"

	"helm.sh/helm/v4/pkg/cli"
)

func TestLoadInstallable(t *testing.T) {
	t.Parallel()

	envSettings := cli.New()
	goodChart := filepath.Join("..", "testdata", "charts", "test-release")
	badChart := filepath.Join("..", "testdata", "charts", "library-release")

	result, err := LoadInstallable(envSettings, goodChart, "0.1.0")
	if err != nil {
		t.Fatalf("expected test chart to load, got error: %v", err)
	}

	if result.Chart == nil {
		t.Fatal("expected loaded chart, got nil")
	}

	if result.Chart.Metadata.Name != "test-release" {
		t.Fatalf("expected chart name %q, got %q", "test-release", result.Chart.Metadata.Name)
	}

	if _, err := LoadInstallable(envSettings, badChart, "0.1.0"); err == nil {
		t.Fatal("expected non-application chart to fail validation")
	}
}