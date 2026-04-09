package helm

import (
	"errors"
	"io"
	"strings"

	"gopkg.in/yaml.v3"

	releasecommon "helm.sh/helm/v4/pkg/release/common"
	helmrelease "helm.sh/helm/v4/pkg/release/v1"
)

// DecodeManifest converts a rendered Helm manifest into decoded YAML documents.
func DecodeManifest(manifest string) ([]any, error) {
	var rendered []any
	decoder := yaml.NewDecoder(strings.NewReader(manifest))

	for {
		var document any
		if err := decoder.Decode(&document); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, err
		}

		if document == nil {
			continue
		}

		rendered = append(rendered, document)
	}

	return rendered, nil
}

// ReleaseStatus returns the string status for a Helm release.
func ReleaseStatus(rel *helmrelease.Release) string {
	if rel == nil || rel.Info == nil {
		return releasecommon.StatusUnknown.String()
	}

	return rel.Info.Status.String()
}
