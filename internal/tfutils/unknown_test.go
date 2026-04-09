package tfutils

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestIsFullyKnown(t *testing.T) {
	t.Parallel()

	knownObject, diags := types.ObjectValue(
		map[string]attr.Type{"name": types.StringType},
		map[string]attr.Value{"name": types.StringValue("helm")},
	)
	if diags.HasError() {
		t.Fatalf("failed to build known object: %v", diags.Errors())
	}

	unknownObject, diags := types.ObjectValue(
		map[string]attr.Type{"name": types.StringType},
		map[string]attr.Value{"name": types.StringUnknown()},
	)
	if diags.HasError() {
		t.Fatalf("failed to build unknown object: %v", diags.Errors())
	}

	tests := []struct {
		name     string
		value    attr.Value
		expected bool
	}{
		{
			name:     "nil_value",
			value:    nil,
			expected: true,
		},
		{
			name:     "known_string",
			value:    types.StringValue("helm"),
			expected: true,
		},
		{
			name:     "unknown_string",
			value:    types.StringUnknown(),
			expected: false,
		},
		{
			name:     "dynamic_known_object",
			value:    types.DynamicValue(knownObject),
			expected: true,
		},
		{
			name:     "dynamic_unknown_object_attribute",
			value:    types.DynamicValue(unknownObject),
			expected: false,
		},
		{
			name: "tuple_with_unknown_element",
			value: types.TupleValueMust(
				[]attr.Type{types.StringType, types.StringType},
				[]attr.Value{types.StringValue("helm"), types.StringUnknown()},
			),
			expected: false,
		},
	}

	for i := range tests {
		testCase := tests[i]

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if actual := IsFullyKnown(testCase.value); actual != testCase.expected {
				t.Fatalf("expected %t, got %t", testCase.expected, actual)
			}
		})
	}
}
