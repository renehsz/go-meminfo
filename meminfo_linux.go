//go:build linux

// SPDX-License-Identifier: MIT
package meminfo

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"syscall"
)

func getMemInfo() (*MemInfo, error) {
	// Try /proc/meminfo first
	memInfo, err := getFromProcMemInfo()
	if err == nil {
		return memInfo, nil
	}
	// Fall back to syscall.Sysinfo
	return getFromSysinfo()
}

//
// /proc/meminfo-based implementation
//

// memVars holds the parsed values from /proc/meminfo
type memVars struct {
	MemTotal     uint64
	MemFree      uint64
	MemAvailable optionalUint64
	Buffers      uint64
	Cached       uint64
	Shmem        uint64
}

func getFromProcMemInfo() (*MemInfo, error) {
	vars, err := getProcMeminfoVars()
	if err != nil {
		return nil, err
	}

	// In the rare case that MemAvailable is not present (e.g. old kernels < 3.14),
	// we can calculate it approximately as MemFree + Buffers + Cached - Shmem.
	if !vars.MemAvailable.Present {
		return &MemInfo{
			Total:     vars.MemTotal,
			Free:      vars.MemFree,
			Available: vars.MemFree + vars.Buffers + vars.Cached - vars.Shmem,
		}, nil
	}

	// If MemAvailable is present, we can use it directly.
	return &MemInfo{
		Total:     vars.MemTotal,
		Free:      vars.MemFree,
		Available: vars.MemAvailable.Value,
	}, nil
}

func getProcMeminfoVars() (*memVars, error) {
	var vars memVars
	rv := reflect.ValueOf(&vars)
	// Read from /proc/meminfo
	if err := readFileVarsIntoStruct("/proc/meminfo", parseLineFromProcMeminfo, rv); err != nil {
		return nil, err
	}
	return &vars, nil
}

// parseLineFromProcMeminfo parses a line from /proc/meminfo and returns the file variable (key and value in bytes or pages).
func parseLineFromProcMeminfo(line string) ([]fileVar, error) {
	lineParts := strings.SplitN(line, ":", 2)
	if len(lineParts) != 2 {
		return nil, fmt.Errorf("invalid line format: \"%s\"; couldn't split line", line)
	}

	key := strings.TrimSpace(lineParts[0])
	valueParts := strings.SplitN(strings.TrimSpace(lineParts[1]), " ", 2)
	if len(valueParts) < 1 || len(valueParts) > 2 {
		return nil, fmt.Errorf("invalid value format: \"%s\"; couldn't parse value", line)
	}

	value, err := strconv.ParseUint(strings.TrimSpace(valueParts[0]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid value format: \"%s\"; couldn't parse value as integer: %v", line, err)
	}
	var valueUnit uint64 = 1
	valueUnitStr := strings.TrimSpace(valueParts[1])
	if strings.EqualFold(valueUnitStr, "kB") {
		valueUnit = 1 << 10
	} else if strings.EqualFold(valueUnitStr, "MB") {
		valueUnit = 1 << 20
	} else if strings.EqualFold(valueUnitStr, "GB") {
		valueUnit = 1 << 30
	} else if strings.EqualFold(valueUnitStr, "TB") {
		valueUnit = 1 << 40
	}
	value = value * valueUnit

	return []fileVar{
		fileVar { key, value },
	}, nil
}

//
// syscall.Sysinfo-based fallback
//

// getFromSysinfo retrieves memory information using syscall.Sysinfo.
// Unfortunately, this method is not accurate at all, as the syscall does not
// provide the same level of detail as /proc/meminfo. It is used as a fallback
// only for the odd case where /proc is not mounted.
func getFromSysinfo() (*MemInfo, error) {
	fmt.Println("meminfo warning: Using syscall.Sysinfo to gather memory information. This is not accurate! Please make sure /proc is available.")
	var sysinfo syscall.Sysinfo_t
	if err := syscall.Sysinfo(&sysinfo); err != nil {
		return nil, err
	}
	return &MemInfo{
		Total: uint64(sysinfo.Totalram) * uint64(sysinfo.Unit),
		Free:  uint64(sysinfo.Freeram) * uint64(sysinfo.Unit),
		// Unfortunately, sysinfo is missing Cached, so we can't do an accurate calculation.
		Available: (uint64(sysinfo.Freeram) + uint64(sysinfo.Bufferram)) * uint64(sysinfo.Unit),
	}, nil
}
