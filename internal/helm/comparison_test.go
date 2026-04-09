package helm

import (
	"strings"
	"testing"

	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	helmrelease "helm.sh/helm/v4/pkg/release/v1"
)

func TestNeedsUpgrade(t *testing.T) {
	t.Parallel()

	newRelease := func() *helmrelease.Release {
		return &helmrelease.Release{
			Name:      "example",
			Namespace: "default",
			Manifest:  "\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: example\n",
			Config: map[string]any{
				"replicas": 1,
			},
			Chart: &chartv2.Chart{
				Metadata: &chartv2.Metadata{
					Name:    "chart",
					Version: "1.0.0",
				},
			},
			Hooks: []*helmrelease.Hook{{
				Name:     "example-hook",
				Manifest: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: hook\n",
			}},
		}
	}

	tests := []struct {
		name          string
		mutateCurrent func(*helmrelease.Release)
		mutatePlanned func(*helmrelease.Release)
		expectUpgrade bool
	}{
		{name: "equivalent_release_does_not_upgrade", expectUpgrade: false},
		{
			name: "manifest_whitespace_is_ignored",
			mutateCurrent: func(rel *helmrelease.Release) {
				rel.Manifest = strings.TrimSpace(rel.Manifest)
			},
			expectUpgrade: false,
		},
		{
			name: "manifest_change_requires_upgrade",
			mutatePlanned: func(rel *helmrelease.Release) {
				rel.Manifest += "\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: example\n"
			},
			expectUpgrade: true,
		},
		{
			name: "config_change_requires_upgrade",
			mutatePlanned: func(rel *helmrelease.Release) {
				rel.Config["replicas"] = 2
			},
			expectUpgrade: true,
		},
		{
			name: "chart_change_requires_upgrade",
			mutatePlanned: func(rel *helmrelease.Release) {
				rel.Chart.Metadata.Version = "1.1.0"
			},
			expectUpgrade: true,
		},
		{
			name: "hook_change_requires_upgrade",
			mutatePlanned: func(rel *helmrelease.Release) {
				rel.Hooks[0].Manifest += "# changed"
			},
			expectUpgrade: true,
		},
		{name: "nil_current_requires_upgrade", expectUpgrade: true},
	}

	for i := range tests {
		testCase := tests[i]

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			current := newRelease()
			planned := newRelease()

			if testCase.mutateCurrent != nil {
				testCase.mutateCurrent(current)
			}

			if testCase.mutatePlanned != nil {
				testCase.mutatePlanned(planned)
			}

			if testCase.name == "nil_current_requires_upgrade" {
				current = nil
			}

			if actual := NeedsUpgrade(current, planned); actual != testCase.expectUpgrade {
				t.Fatalf("expected upgrade=%t, got %t", testCase.expectUpgrade, actual)
			}
		})
	}
}
