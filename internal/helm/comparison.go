package helm

import (
	"reflect"
	"strings"

	helmrelease "helm.sh/helm/v4/pkg/release/v1"
)

// NeedsUpgrade returns true when a real Helm upgrade is required.
func NeedsUpgrade(current, planned *helmrelease.Release) bool {
	if current == nil || planned == nil {
		return true
	}

	if NormalizeManifest(current.Manifest) != NormalizeManifest(planned.Manifest) {
		return true
	}

	if !reflect.DeepEqual(current.Config, planned.Config) {
		return true
	}

	if !reflect.DeepEqual(current.Chart, planned.Chart) {
		return true
	}

	if !reflect.DeepEqual(current.Hooks, planned.Hooks) {
		return true
	}

	return false
}

// NormalizeManifest removes inconsequential surrounding whitespace before comparison.
func NormalizeManifest(manifest string) string {
	return strings.TrimSpace(manifest)
}
