//go:build freebsd

// SPDX-License-Identifier: MIT
package meminfo

import (
	"fmt"
	"reflect"

	// Unfortunately, the standard library's syscall package does not work
	// for our purposes, as it doesn't expose a way to retrieve a 64-bit
	// integer from sysctl. This third-party dependency let's us do just
	// that but depends on libc unfortunately.
	// TODO: Replace this dependency with our own implementation which
	//       will probably have to be based on syscall.RawSyscall().
	"github.com/blabber/go-freebsd-sysctl/sysctl"
)

func getMemInfo() (*MemInfo, error) {
	var vars memVars
	if err := getVarsFromSysctl(reflect.ValueOf(&vars)); err != nil {
		return nil, err
	}
	return &MemInfo{
		Total:     vars.HwPhysMem,
		Free:      vars.VmFreeCount * vars.HwPageSize,
		Available: (vars.VmFreeCount + vars.VmInactiveCount + vars.VmCacheCount) * vars.HwPageSize,
	}, nil
}

type memVars struct {
	HwPhysMem       uint64 `sysctl:"hw.physmem"`
	HwPageSize      uint64 `sysctl:"hw.pagesize"`
	VmPageCount     uint64 `sysctl:"vm.stats.vm.v_page_count"`
	VmWireCount     uint64 `sysctl:"vm.stats.vm.v_wire_count"`
	VmActiveCount   uint64 `sysctl:"vm.stats.vm.v_active_count"`
	VmInactiveCount uint64 `sysctl:"vm.stats.vm.v_inactive_count"`
	VmCacheCount    uint64 `sysctl:"vm.stats.vm.v_cache_count"`
	VmFreeCount     uint64 `sysctl:"vm.stats.vm.v_free_count"`
}

// getVarsFromSysctl looks up all the fields of the struct in the first
// argument and tries to retrieve the corresponding values from sysctl.
func getVarsFromSysctl(rv reflect.Value) error {
	rvi := reflect.Indirect(rv)
	numFieldsLeft := rvi.NumField()
	for i := 0; i < rvi.NumField(); i++ {
		field := rvi.Field(i)
		// use the field's `sysctl:` tag
		name, found := rvi.Type().Field(i).Tag.Lookup("sysctl")
		if !found {
			panic(fmt.Sprintf("struct field %s is missing `sysctl:` tag", rvi.Type().Field(i).Name))
		}

		// call sysctl(3)
		value, err := sysctl.GetInt64(name)

		// save the result variable into our data structure
		numFieldsLeft--
		if field.Kind() == reflect.Struct {
			if err == nil {
				field.Set(reflect.ValueOf(optionalUint64{Present: true, Value: uint64(value)}))
			} else {
				field.Set(reflect.ValueOf(optionalUint64{Present: false, Value: 0}))
			}
		} else if field.Kind() == reflect.Uint64 {
			// for now, we just bail out in case a mandatory variable
			// (declared as uint64 in the struct that's passed in) is
			// missing or invalid
			if err != nil {
				return fmt.Errorf("sysctl(\"%s\") failed: %v", name, err)
			}
			field.Set(reflect.ValueOf(uint64(value)))
		} else {
			panic("invalid field type")
		}
	}
	return nil
}

