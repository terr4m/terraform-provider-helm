package helm

import "testing"

func TestDecodeManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		manifest      string
		expectError   bool
		expectedCount int
	}{
		{
			name: "multiple_documents",
			manifest: `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: first
---
apiVersion: v1
kind: Service
metadata:
  name: second
`,
			expectError:   false,
			expectedCount: 2,
		},
		{name: "empty_manifest", manifest: "", expectError: false, expectedCount: 0},
		{name: "invalid_yaml", manifest: "[not valid yaml", expectError: true, expectedCount: 0},
	}

	for i := range tests {
		testCase := tests[i]

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rendered, err := DecodeManifest(testCase.manifest)
			if (err != nil) != testCase.expectError {
				t.Fatalf("expected error=%t, got %v", testCase.expectError, err)
			}

			if len(rendered) != testCase.expectedCount {
				t.Fatalf("expected %d decoded documents, got %d", testCase.expectedCount, len(rendered))
			}
		})
	}
}