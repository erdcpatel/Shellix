package web

import (
	"runtime"
	"sync"
	"time"
)

// ProcessMetrics tracks current resource consumption of the Shellix process
type ProcessMetrics struct {
	CPUPercent float64 `json:"cpu_percent"`
	AllocMB    float64 `json:"alloc_mb"`
	SysMB      float64 `json:"sys_mb"`
	Goroutines int     `json:"goroutines"`
	NumGC      uint32  `json:"num_gc"`
	Cores      int     `json:"cores"`
	UptimeSec  int64   `json:"uptime_sec"`
}

type metricsCollector struct {
	mu           sync.Mutex
	startTime    time.Time
	lastSample   time.Time
	lastCPUTime  time.Duration
	lastCPUUsage float64
}

var defaultCollector = &metricsCollector{
	startTime:   time.Now(),
	lastSample:  time.Now(),
	lastCPUTime: getProcessCPUTime(),
}

// SampleMetrics collects the latest CPU, memory, and goroutine counts
func SampleMetrics() ProcessMetrics {
	defaultCollector.mu.Lock()
	defer defaultCollector.mu.Unlock()

	now := time.Now()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	cpuTime := getProcessCPUTime()
	elapsedWall := now.Sub(defaultCollector.lastSample)

	var cpuPercent float64
	if elapsedWall > 80*time.Millisecond && defaultCollector.lastCPUTime > 0 {
		cpuDelta := cpuTime - defaultCollector.lastCPUTime
		if cpuDelta >= 0 {
			percent := (float64(cpuDelta) / float64(elapsedWall)) * 100
			if percent < 0 {
				percent = 0
			}
			cpuPercent = percent
			defaultCollector.lastCPUUsage = percent
			defaultCollector.lastCPUTime = cpuTime
			defaultCollector.lastSample = now
		} else {
			cpuPercent = defaultCollector.lastCPUUsage
		}
	} else {
		cpuPercent = defaultCollector.lastCPUUsage
	}

	allocMB := float64(mem.Alloc) / (1024 * 1024)
	sysMB := float64(mem.Sys) / (1024 * 1024)
	goroutines := runtime.NumGoroutine()

	return ProcessMetrics{
		CPUPercent: cpuPercent,
		AllocMB:    allocMB,
		SysMB:      sysMB,
		Goroutines: goroutines,
		NumGC:      mem.NumGC,
		Cores:      runtime.NumCPU(),
		UptimeSec:  int64(time.Since(defaultCollector.startTime).Seconds()),
	}
}
