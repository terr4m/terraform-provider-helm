package tfutils

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DecodeDynamic decodes an object into a Terraform dynamic value.
func DecodeDynamic(ctx context.Context, obj any) (types.Dynamic, diag.Diagnostics) {
	if obj == nil {
		return types.DynamicNull(), nil
	}

	val, diags := decodeScalar(ctx, obj)
	if diags.HasError() {
		return types.Dynamic{}, diags
	}

	return types.DynamicValue(val), diags
}

// decodeScalar decodes a scalar value into a Terraform attribute value.
func decodeScalar(ctx context.Context, a any) (attr.Value, diag.Diagnostics) {
	switch v := a.(type) {
	case nil:
		return types.DynamicNull(), nil
	case int64:
		return types.NumberValue(big.NewFloat(float64(v))), nil
	case float64:
		return types.NumberValue(big.NewFloat(v)), nil
	case bool:
		return types.BoolValue(v), nil
	case string:
		return types.StringValue(v), nil
	case []any:
		return decodeSlice(ctx, v)
	case map[string]any:
		return decodeMap(ctx, v)
	default:
		diagnostics := diag.Diagnostics{}
		diagnostics.AddError("Unexpected type.", fmt.Sprintf("unexpected type: %T for value %#v", v, v))
		return nil, diagnostics
	}
}

// decodeSlice decodes a sequence value into a Terraform attribute value.
func decodeSlice(ctx context.Context, s []any) (attr.Value, diag.Diagnostics) {
	l := len(s)
	vl := make([]attr.Value, 0, l)
	tl := make([]attr.Type, 0, l)

	for _, v := range s {
		vv, err := decodeScalar(ctx, v)
		if err != nil {
			return nil, err
		}

		if vv != nil {
			vl = append(vl, vv)
			tl = append(tl, vv.Type(ctx))
		}
	}

	if len(vl) > 0 {
		return types.TupleValue(tl, vl)
	}

	return nil, nil
}

// decodeMap decodes a mapping value into a Terraform attribute value.
func decodeMap(ctx context.Context, m map[string]any) (attr.Value, diag.Diagnostics) {
	l := len(m)
	vm := make(map[string]attr.Value, l)
	tm := make(map[string]attr.Type, l)

	for k, v := range m {
		vv, diags := decodeScalar(ctx, v)
		if diags.HasError() {
			return nil, diags
		}

		if vv != nil {
			vm[k] = vv
			tm[k] = vv.Type(ctx)
		}
	}

	if len(vm) > 0 {
		return types.ObjectValue(tm, vm)
	}

	return nil, nil
}
