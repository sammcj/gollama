package cuda

// import (
// 	"fmt"

// 	"github.com/NVIDIA/go-nvml/pkg/nvml"
// )

// func GetCUDAVRAM() (float64, error) {
// 	ret := nvml.Init()
// 	if ret != nvml.SUCCESS {
// 		return 0, fmt.Errorf("failed to initialize NVML: %v", ret)
// 	}
// 	defer nvml.Shutdown()

// 	count, ret := nvml.DeviceGetCount()
// 	if ret != nvml.SUCCESS {
// 		return 0, fmt.Errorf("failed to get device count: %v", ret)
// 	}

// 	var totalVRAM uint64
// 	for i := 0; i < int(count); i++ {
// 		device, ret := nvml.DeviceGetHandleByIndex(int(i))
// 		if ret != nvml.SUCCESS {
// 			return 0, fmt.Errorf("failed to get device handle for device %d: %v", i, ret)
// 		}

// 		memory, ret := device.GetMemoryInfo()
// 		if ret != nvml.SUCCESS {
// 			return 0, fmt.Errorf("failed to get memory info for device %d: %v", i, ret)
// 		}

// 		totalVRAM += memory.Total
// 	}

// 	// Convert to GB
// 	totalVRAMGB := float64(totalVRAM) / (1024 * 1024 * 1024)

// 	return totalVRAMGB, nil
// }
