package core

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	. "mcsd/utils"
)

type HostVitals struct {
	MemoryTotal int     `json:"memory_total"`
	MemoryUsed  int     `json:"memory_used"`
	LoadAvg1    float64 `json:"load_avg_1"`
	CPUCores    int     `json:"cpu_cores"`
	DiskTotalGB int     `json:"disk_total_gb"`
	DiskUsedGB  int     `json:"disk_used_gb"`
}

type BudgetVitals struct {
	Total int `json:"total"`
	Used  int `json:"used"`
}

func ReadHostVitals() (HostVitals, error) {
	memTotal, memAvail, err := readMemInfo()
	if err != nil {
		return HostVitals{}, err
	}
	loadAvg, err := readLoadAvg()
	if err != nil {
		return HostVitals{}, err
	}
	diskTotal, diskFree, err := readDiskStats()
	if err != nil {
		return HostVitals{}, err
	}
	return HostVitals{
		MemoryTotal: int(memTotal / 1024),
		MemoryUsed:  int((memTotal - memAvail) / 1024),
		LoadAvg1:    loadAvg,
		CPUCores:    runtime.NumCPU(),
		DiskTotalGB: int(diskTotal / (1024 * 1024 * 1024)),
		DiskUsedGB:  int((diskTotal - diskFree) / (1024 * 1024 * 1024)),
	}, nil
}

func ReadBudgetVitals() (BudgetVitals, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return BudgetVitals{}, &InternalError{Message: "global config not loaded"}
	}
	used, err := TotalReservedMemory("")
	if err != nil {
		return BudgetVitals{}, err
	}
	return BudgetVitals{Total: cfg.MemoryBudget, Used: used}, nil
}

func readMemInfo() (total, available uint64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		v, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = v
		case "MemAvailable:":
			available = v
		}
	}
	return total, available, scanner.Err()
}

func readLoadAvg() (float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, &InternalError{Message: "empty /proc/loadavg"}
	}
	return strconv.ParseFloat(fields[0], 64)
}

func readDiskStats() (total, free uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/srv/mcsd", &stat); err != nil {
		return 0, 0, err
	}
	bsize := uint64(stat.Bsize)
	return stat.Blocks * bsize, stat.Bavail * bsize, nil
}
