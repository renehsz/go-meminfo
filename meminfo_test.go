// SPDX-License-Identifier: MIT
package meminfo

import (
	"reflect"
	"testing"
)

func TestGetMeminfoNonzero(t *testing.T) {
	m, err := Get()
	if err != nil {
		t.Fatalf("Error getting memory info: %v", err)
	}
	checkFieldsPresentAndNonZero(t, reflect.ValueOf(*m))
}

// checkFieldsPresentAndNonZero is a test helper that checks if all values
// in the given struct are
//  - present and non-zero in case of an optionalUint64 and
//  - non-zero in case of a uint64.
func checkFieldsPresentAndNonZero(t *testing.T, rv reflect.Value) {
	// Check if all fields are set (present and non-zero)
	numFields := rv.NumField()
	for i := 0; i < numFields; i++ {
		field := rv.Field(i)
		if field.Kind() == reflect.Struct {
			ou := field.Interface().(optionalUint64)
			if ou.Present == false || ou.Value == 0 {
				t.Errorf("Expected %s to be present", rv.Type().Field(i).Name)
			}
		} else if field.Kind() == reflect.Uint64 {
			if field.Uint() == 0 {
				t.Errorf("Expected %s to be non-zero", rv.Type().Field(i).Name)
			}
		}
	}
}
