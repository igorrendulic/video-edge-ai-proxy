package services

import (
	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/mem"
)

type hardwareService struct{}

func NewHardwareService() *hardwareService {
	return &hardwareService{}
}

func (hw *hardwareService) GetMemory() error {
	runtimeOS := runtime.GOOS
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	fmt.Printf("runtime os: %v\n", runtimeOS)
	fmt.Printf("vmStat: %v\n", vmStat)

	return nil
}
