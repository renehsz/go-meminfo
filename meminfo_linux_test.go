//go:build linux

// SPDX-License-Identifier: MIT
package meminfo

import (
	"reflect"
	"testing"
)

func TestFallbackMethodsEqual(t *testing.T) {
	memInfo1, _ := getFromProcMemInfo()
	memInfo2, _ := getFromSysinfo()
	if memInfo1 == nil || memInfo2 == nil {
		t.Fatal("Expected non-nil memory info")
	}
	if memInfo1.Total != memInfo2.Total {
		t.Errorf("Mismatched Total: %d != %d", memInfo1.Total, memInfo2.Total)
	}
	if memInfo1.Free != memInfo2.Free {
		t.Errorf("Mismatched Free: %d != %d", memInfo1.Free, memInfo2.Free)
	}
	// Available from getFromSysinfo is not expected to be accurate,
	// so we skip this check
}

func TestGetFromSysinfo(t *testing.T) {
	memInfo, err := getFromSysinfo()
	if err != nil {
		t.Fatalf("Error getting memory info: %v", err)
	}
	if memInfo == nil {
		t.Fatal("Expected non-nil memory info")
	}
	checkFieldsPresentAndNonZero(t, reflect.ValueOf(*memInfo))
}

func TestGetProcMemInfoVars(t *testing.T) {
	vars, err := getProcMeminfoVars()
	if err != nil {
		t.Fatalf("Error getting memory info: %v", err)
	}
	if vars == nil {
		t.Fatal("Expected non-nil memory info")
	}
	checkFieldsPresentAndNonZero(t, reflect.ValueOf(*vars))
}

