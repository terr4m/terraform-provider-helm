package tfutils

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEncodeDynamicObject(t *testing.T) {
	t.Parallel()

	simpleObject, _ := types.ObjectValue(map[string]attr.Type{"foo": types.StringType}, map[string]attr.Value{"foo": types.StringValue("bar")})

	for _, d := range []struct {
		testName string
		dyn      types.Dynamic
		expected map[string]any
		errMsg   string
	}{
		{
			testName: "invalid",
			dyn:      types.DynamicValue(types.StringValue("foo")),
			expected: nil,
			errMsg:   "expected object value, got basetypes.StringValue",
		},
		{
			testName: "object",
			dyn:      types.DynamicValue(simpleObject),
			expected: map[string]any{"foo": "bar"},
			errMsg:   "",
		},
	} {
		t.Run(d.testName, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			actual, err := EncodeDynamicObject(ctx, d.dyn)

			if !reflect.DeepEqual(actual, d.expected) {
				t.Errorf("expected %v, got %v", d.expected, actual)
			}

			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}

			if errMsg != d.errMsg {
				t.Errorf("expected error message %s, got %s", d.errMsg, errMsg)
			}
		})
	}
}
