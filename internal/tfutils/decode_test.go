package tfutils

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDecodeDynamic(t *testing.T) {
	t.Parallel()

	simpleObject, _ := types.ObjectValue(map[string]attr.Type{"foo": types.StringType}, map[string]attr.Value{"foo": types.StringValue("bar")})
	stringTuple, _ := types.TupleValue([]attr.Type{types.StringType, types.StringType}, []attr.Value{types.StringValue("foo"), types.StringValue("bar")})

	for _, d := range []struct {
		testName string
		obj      any
		expected types.Dynamic
		errMsg   string
	}{
		{
			testName: "unexpected_type",
			obj:      1,
			expected: types.Dynamic{},
			errMsg:   "Unexpected type.",
		},
		{
			testName: "null",
			obj:      nil,
			expected: types.DynamicNull(),
			errMsg:   "",
		},
		{
			testName: "int64",
			obj:      int64(1),
			expected: types.DynamicValue(types.NumberValue(big.NewFloat(float64(1)))),
			errMsg:   "",
		},
		{
			testName: "float64",
			obj:      float64(1.1),
			expected: types.DynamicValue(types.NumberValue(big.NewFloat(1.1))),
			errMsg:   "",
		},
		{
			testName: "bool",
			obj:      true,
			expected: types.DynamicValue(types.BoolValue(true)),
			errMsg:   "",
		},
		{
			testName: "string",
			obj:      "foo",
			expected: types.DynamicValue(types.StringValue("foo")),
			errMsg:   "",
		},
		{
			testName: "object_empty",
			obj:      map[string]any{},
			expected: types.DynamicNull(),
		},
		{
			testName: "object_simple",
			obj:      map[string]any{"foo": "bar"},
			expected: types.DynamicValue(simpleObject),
			errMsg:   "",
		},
		{
			testName: "array_empty",
			obj:      []any{},
			expected: types.DynamicNull(),
			errMsg:   "",
		},
		{
			testName: "array_strings",
			obj:      []any{"foo", "bar"},
			expected: types.DynamicValue(stringTuple),
			errMsg:   "",
		},
	} {
		t.Run(d.testName, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			dyn, diags := DecodeDynamic(ctx, d.obj)

			if !dyn.Equal(d.expected) {
				t.Errorf("expected %v, got %v", d.expected, dyn)
			}

			var errMsg string
			if diags.HasError() {
				for i, diag := range diags.Errors() {
					if i == 0 {
						errMsg = diag.Summary()
						continue
					}
					errMsg = fmt.Sprintf("%s: %s", errMsg, diag.Summary())
				}
			}

			if errMsg != d.errMsg {
				t.Errorf("expected error message %s, got %s", d.errMsg, errMsg)
			}
		})
	}
}
