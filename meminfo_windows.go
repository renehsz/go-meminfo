//go:build windows

// SPDX-License-Identifier: MIT
package meminfo

import (
	"syscall"
	"unsafe"
)

func getMemInfo() (*MemInfo, error) {
	memStatus, err := globalMemoryStatusEx()
	if err != nil {
		return nil, err
	}

	return &MemInfo{
		Total: memStatus.TotalPhys,
		Free: memStatus.AvailPhys,
		Available: memStatus.AvailPhys, // TODO: this is inaccurate, it doesn't include things like VFS caches that can easily be flushed if needed
	}, nil
}

var kernel32_dll = syscall.NewLazyDLL("kernel32.dll")
var globalMemoryStatusExProc = kernel32_dll.NewProc("GlobalMemoryStatusEx")

type memoryStatusEx struct {
	StructLength         uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func globalMemoryStatusEx() (*memoryStatusEx, error) {
	var buffer memoryStatusEx
	buffer.StructLength = uint32(unsafe.Sizeof(buffer))

	// call GlobalMemoryStatusEx()
	success, _, lastErr := globalMemoryStatusExProc.Call(uintptr(unsafe.Pointer(&buffer)))
	// NOTE: lastErr is always the result of GetLastError, even in case of success
	if success == 0 {
		return nil, lastErr
	}

	return &buffer, nil
}

