package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestHelmProviderResources(t *testing.T) {
	t.Parallel()

	p := &HelmProvider{}
	resources := p.Resources(context.Background())
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}

	r, ok := resources[0]().(*ReleaseResource)
	if !ok {
		t.Fatalf("expected %T, got %T", &ReleaseResource{}, resources[0]())
	}

	if r == nil {
		t.Fatal("expected release resource instance, got nil")
	}
}

func TestReleaseResourceConfigure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		providerData        any
		expectDiagnostics   bool
		expectedSummary     string
		expectStoredPointer bool
	}{
		{
			name:                "nil_provider_data",
			providerData:        nil,
			expectDiagnostics:   false,
			expectStoredPointer: false,
		},
		{
			name:                "unexpected_provider_data_type",
			providerData:        "nope",
			expectDiagnostics:   true,
			expectedSummary:     "Unexpected resource provider data.",
			expectStoredPointer: false,
		},
		{
			name:                "stores_provider_data",
			providerData:        &HelmProviderData{},
			expectDiagnostics:   false,
			expectStoredPointer: true,
		},
	}

	for i := range tests {
		testCase := tests[i]

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			r := &ReleaseResource{}
			resp := &resource.ConfigureResponse{}

			r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: testCase.providerData}, resp)

			if resp.Diagnostics.HasError() != testCase.expectDiagnostics {
				t.Fatalf("expected diagnostics=%t, got %t", testCase.expectDiagnostics, resp.Diagnostics.HasError())
			}

			if testCase.expectDiagnostics {
				errs := resp.Diagnostics.Errors()
				if len(errs) != 1 {
					t.Fatalf("expected 1 diagnostic error, got %d", len(errs))
				}

				if errs[0].Summary() != testCase.expectedSummary {
					t.Fatalf("expected diagnostic summary %q, got %q", testCase.expectedSummary, errs[0].Summary())
				}
			}

			providerData, _ := testCase.providerData.(*HelmProviderData)
			if testCase.expectStoredPointer {
				if r.providerData != providerData {
					t.Fatal("expected provider data to be stored on the resource")
				}
				if providerData != nil && providerData.Client != nil && r.service == nil {
					t.Fatal("expected release service to be initialized when provider client is available")
				}
				return
			}

			if r.providerData != nil {
				t.Fatal("expected provider data to remain nil")
			}
		})
	}
}
