//go:build linux

// SPDX-License-Identifier: MIT
package meminfo

import (
	"bufio"
	"fmt"
	"os"
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

type optionalUint64 struct {
	Present bool
	Value   uint64
}

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
	// Read /proc/meminfo
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		// TODO: any fallbacks?
		return nil, err
	}
	defer f.Close()
	var vars memVars
	rv := reflect.ValueOf(&vars)
	numFieldsLeft := reflect.Indirect(rv).NumField()

	// Parse line by line
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if numFieldsLeft == 0 {
			break
		}

		key, value, err := parseLine(scanner.Text())
		if err != nil {
			continue
		}

		// Check if the key is one of the fields in memVars and set the value accordingly
		for i := 0; i < reflect.Indirect(rv).NumField(); i++ {
			fieldName := reflect.Indirect(rv).Type().Field(i).Name
			field := reflect.Indirect(rv).Field(i)
			if key == fieldName {
				numFieldsLeft--
				if field.Kind() == reflect.Struct {
					field.Set(reflect.ValueOf(optionalUint64{Present: true, Value: value}))
				} else if field.Kind() == reflect.Uint64 {
					field.Set(reflect.ValueOf(value))
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &vars, nil
}

// parseLine parses a line from /proc/meminfo and returns the key and value in bytes.
func parseLine(line string) (string, uint64, error) {
	lineParts := strings.SplitN(line, ":", 2)
	if len(lineParts) != 2 {
		return "", 0, fmt.Errorf("invalid line format: \"%s\"; couldn't split line", line)
	}

	key := strings.TrimSpace(lineParts[0])
	valueParts := strings.SplitN(strings.TrimSpace(lineParts[1]), " ", 2)
	if len(valueParts) < 1 || len(valueParts) > 2 {
		return "", 0, fmt.Errorf("invalid value format: \"%s\"; couldn't parse value", line)
	}

	value, err := strconv.ParseUint(strings.TrimSpace(valueParts[0]), 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid value format: \"%s\"; couldn't parse value as integer: %v", line, err)
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

	return key, value, nil
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
