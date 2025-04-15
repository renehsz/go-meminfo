//go:build plan9

// SPDX-License-Identifier: MIT
package meminfo

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func getMemInfo() (*MemInfo, error) {
	vars, err := getDevSwapVars()
	if err != nil {
		return nil, err
	}
	return &MemInfo{
		Total:     vars.Memory,
		Free:      vars.UserAvailable * vars.Pagesize,
		// TODO: What about `reclaim`?
		//       It's not documented in swap(3) :-(, so it might be new
		Available: vars.UserAvailable * vars.Pagesize,
	}, nil
}

// devSwapVars holds the parsed values from /dev/swap
type devSwapVars struct {
	Memory                 uint64
	Pagesize               uint64
	UserUsed               uint64
	UserAvailable          uint64
	SwapUsed               uint64
	SwapAvailable          uint64
	ReclaimUsed            optionalUint64
	ReclaimAvailable       optionalUint64
	KernelMallocAllocation uint64
	KernelMallocUsed       uint64
	KernelMallocAvailable  uint64
	KernelDrawAllocation   uint64
	KernelDrawUsed         uint64
	KernelDrawAvailable    uint64
	KernelSecretAllocation uint64
	KernelSecretUsed       uint64
	KernelSecretAvailable  uint64
}

func getDevSwapVars() (*devSwapVars, error) {
	var vars devSwapVars
	rv := reflect.ValueOf(&vars)
	// Read from /dev/swap
	if err := readFileVarsIntoStruct("/dev/swap", parseLineFromDevSwap, rv); err != nil {
		return nil, err
	}
	return &vars, nil
}

// parseLineFromDevSwap parses a single line from the /dev/swap device
// into a slice of file variables.
//
// Keys are converted from "lower space case" to "UpperCamelCase".
//
// Values can follow one of the following formats
//  1. `n` means `n` bytes or pages,
//  2. `n/m` means `n` *used* out of `m` *available*,
//  3. `a/n/m` means `a` *a*, `n` *used* out of `m` *available*.
// (see swap(3) for more details)
//
// For 1), we just return the file variable directly.
//
// For 2) and 3), we return multiple file variables, where
// used, available, or a is appended to the corresponding keys.
//
// For instance, the line
// ```
// 768/65568/16777216 kernel secret`
// ```
// will be turned into the following result:
// - KernelSecretAllocation: 768,
// - KernelSecretUsed: 65568,
// - KernelSecretAvailable: 16777216.
//
func parseLineFromDevSwap(line string) ([]fileVar, error) {
	lineParts := strings.SplitN(line, " ", 2)
	if len(lineParts) != 2 {
		return nil, fmt.Errorf("invalid line format: \"%s\"; couldn't split line", line)
	}

	key := strings.TrimSpace(lineParts[1])
	key = convertLowerSpaceToUpperCamelCase(key)

	valueStrs := strings.Split(strings.TrimSpace(lineParts[0]), "/")
	values := make([]uint64, len(valueStrs))
	
	for i, valueStr := range valueStrs {
		var err error
		values[i], err = strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	switch len(values) {
		case 1:
			return []fileVar{
				fileVar{
					Key: key,
					Value: values[0],
				},
			}, nil
		case 2:
			return []fileVar{
				fileVar{
					Key: key + "Used",
					Value: values[0],
				},
				fileVar{
					Key: key + "Available",
					Value: values[1],
				},
			}, nil
		case 3:
			return []fileVar{
				fileVar{
					Key: key + "Allocation",
					Value: values[0],
				},
				fileVar{
					Key: key + "Used",
					Value: values[1],
				},
				fileVar{
					Key: key + "Available",
					Value: values[2],
				},
			}, nil
		default:
			return nil, fmt.Errorf("invalid value format: \"%s\"; couldn't parse value", line)
	}
}

// convertLowerSpaceToUpperCamelCase converts `lower space case` to `UpperCamelCase`.
func convertLowerSpaceToUpperCamelCase(s string) string {
	var (
		result string
		curr, prev rune
	)
	for i, r := range s {
		curr = r
		if i == 0 || prev == ' ' {
			result += strings.ToUpper(string(curr))
		} else {
			result += string(curr)
		}
		prev = curr
	}
	return result
}

