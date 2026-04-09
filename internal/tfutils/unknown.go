package tfutils

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IsFullyKnown returns true if the given attribute value is fully known.
func IsFullyKnown(val attr.Value) bool {
	if val == nil {
		return true
	}

	if val.IsUnknown() {
		return false
	}

	switch v := val.(type) {
	case types.Dynamic:
		return IsFullyKnown(v.UnderlyingValue())
	case types.List:
		for _, e := range v.Elements() {
			if !IsFullyKnown(e) {
				return false
			}
		}
		return true
	case types.Set:
		for _, e := range v.Elements() {
			if !IsFullyKnown(e) {
				return false
			}
		}
		return true
	case types.Tuple:
		for _, e := range v.Elements() {
			if !IsFullyKnown(e) {
				return false
			}
		}
		return true
	case types.Map:
		for _, e := range v.Elements() {
			if !IsFullyKnown(e) {
				return false
			}
		}
		return true
	case types.Object:
		for _, e := range v.Attributes() {
			if !IsFullyKnown(e) {
				return false
			}
		}
		return true
	default:
		return true
	}
}
