package vramestimator

import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)


func GetCUDAVRAM() (float64, error) {
	if ret := nvml.Init(); ret != nvml.SUCCESS {
		return 0, fmt.Errorf("failed to initialize NVML: %v", ret)
	}
	defer nvml.Shutdown()

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return 0, fmt.Errorf("failed to get device count: %v", ret)
	}

	var totalVRAM uint64
	for i := 0; i < int(count); i++ {
		device, ret := nvml.DeviceGetHandleByIndex(int(i))
		if ret != nvml.SUCCESS {
			return 0, fmt.Errorf("failed to get device handle: %v", ret)
		}

		memory, ret := device.GetMemoryInfo()
		if ret != nvml.SUCCESS {
			return 0, fmt.Errorf("failed to get memory info: %v", ret)
		}

		totalVRAM += memory.Total
	}

	return float64(totalVRAM) / 1024 / 1024 / 1024, nil // Convert to GB
}
