// File: vramestimator/cuda.go

package vramestimator

import (
	"os"
	"runtime"
)

const (
	CUDASize = 500 * 1024 * 1024 // 500 MB
)

func checkNVMLAvailable() bool {
	if runtime.GOOS == "darwin" {
		return false
	}
	if _, err := os.Stat("/usr/lib/libnvidia-ml.so"); err == nil {
		return true
	}
	return false
}
