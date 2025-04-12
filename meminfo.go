// SPDX-License-Identifier: MIT
package meminfo

type MemInfo struct {
	// Total amount of physical memory installed.
	Total uint64
	// Free memory: This is the amount of memory that is not being used at all
	// (not even for caches or buffers). It is immediately available for
	// allocation.
	Free uint64
	// Available memory: This is an estimate of how much memory is immediately
	// available for allocation by new processes, calculated from unused memory
	// plus reclaimable caches and buffers minus some system overhead.
	Available uint64
}

func Get() (*MemInfo, error) {
	return getMemInfo()
}
