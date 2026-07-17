package api

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"mcsd/core"
)

type hostVitals struct {
	MemoryTotal int     `json:"memory_total"`
	MemoryUsed  int     `json:"memory_used"`
	LoadAvg1    float64 `json:"load_avg_1"`
	CPUCores    int     `json:"cpu_cores"`
	DiskTotalGB int     `json:"disk_total_gb"`
	DiskUsedGB  int     `json:"disk_used_gb"`
}

type budgetVitals struct {
	Total int `json:"total"`
	Used  int `json:"used"`
}

type snapshotResponse struct {
	Host   *hostVitals   `json:"host"`
	Budget *budgetVitals `json:"budget"`
}

func readBudgetVitals() (*budgetVitals, error) {
	cfg := getCachedGlobalConfig()
	if cfg == nil {
		return nil, fmt.Errorf("global config not loaded")
	}
	instances := getCachedInstances()
	var used int
	for id, inst := range instances {
		status, err := sdClient.Status(core.UnitName(id))
		if err != nil || status.State != "active" {
			continue
		}
		used += inst.Memory
	}
	return &budgetVitals{Total: cfg.MemoryBudget, Used: used}, nil
}

var cachedPublicIP string

func initPublicIP() {
	resp, err := http.Get("https://api4.ipify.org?format=text")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	ipBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	cachedPublicIP = strings.TrimSpace(string(ipBytes))
}

func getVitals(w http.ResponseWriter, r *http.Request) {
	host, err := readHostVitals()
	if err != nil {
		writeError(w, err)
		return
	}
	budget, err := readBudgetVitals()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshotResponse{Host: host, Budget: budget})
}

func readHostVitals() (*hostVitals, error) {
	memTotal, memAvail, err := readMemInfo()
	if err != nil {
		return nil, err
	}
	loadAvg, err := readLoadAvg()
	if err != nil {
		return nil, err
	}
	diskTotal, diskFree, err := readDiskStats()
	if err != nil {
		return nil, err
	}
	return &hostVitals{
		MemoryTotal: int(memTotal / 1024),
		MemoryUsed:  int((memTotal - memAvail) / 1024),
		LoadAvg1:    loadAvg,
		CPUCores:    runtime.NumCPU(),
		DiskTotalGB: int(diskTotal / (1024 * 1024 * 1024)),
		DiskUsedGB:  int((diskTotal - diskFree) / (1024 * 1024 * 1024)),
	}, nil
}

// readMemInfo parses /proc/meminfo; returns (MemTotal kB, MemAvailable kB).
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

// readLoadAvg reads the 1-minute load average from /proc/loadavg.
func readLoadAvg() (float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, fmt.Errorf("empty /proc/loadavg")
	}
	return strconv.ParseFloat(fields[0], 64)
}

// readDiskStats returns (total bytes, free bytes) for the mcsd data directory.
func readDiskStats() (total, free uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/srv/mcsd", &stat); err != nil {
		return 0, 0, err
	}
	bsize := uint64(stat.Bsize)
	return stat.Blocks * bsize, stat.Bavail * bsize, nil
}
