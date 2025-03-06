package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Define Prometheus metrics
var (
	serverUptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_server_uptime_seconds",
		Help: "Server uptime in seconds",
	})
	systemLoad = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "game_system_load",
		Help: "System load averages (1m, 5m, 15m)",
	}, []string{"duration"})
	cpuUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_cpu_usage_percent",
		Help: "CPU usage percentage",
	})
	memoryTotalSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_memory_total_size_bytes",
		Help: "Total memory size in bytes",
	})
	memoryUsagePercent = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_memory_usage_percent",
		Help: "Memory usage percentage",
	})
	memoryUsageBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_memory_usage_bytes",
		Help: "Memory usage in bytes",
	})
	memoryFreePercent = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_memory_free_percent",
		Help: "Percentage of free memory",
	})
	memoryFreeBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_memory_free_bytes",
		Help: "Free memory in bytes",
	})
	diskTotalSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_disk_total_size_bytes",
		Help: "Total size of all disks in bytes",
	})
	diskTotalAvailableBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_disk_total_available_bytes",
		Help: "Total available bytes across all disks",
	})
	diskTotalAvailablePercent = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_disk_total_available_percent",
		Help: "Percentage of total disk space available",
	})
	diskTotalUsedBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_disk_total_used_bytes",
		Help: "Total used bytes across all disks",
	})
	diskTotalUsedPercent = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "game_disk_total_used_percent",
		Help: "Percentage of total disk space that is used",
	})
	diskUsagePercent = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "game_disk_usage_percent",
		Help: "Disk usage percentage per partition",
	}, []string{"partition"})
	diskSize = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "game_disk_size_bytes",
		Help: "Total disk size in bytes per partition",
	}, []string{"partition"})
	diskUsed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "game_disk_used_bytes",
		Help: "Used disk space in bytes per partition",
	}, []string{"partition"})
	diskAvailable = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "game_disk_available_bytes",
		Help: "Available disk space in bytes per partition",
	}, []string{"partition"})
	diskPerformance = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "game_disk_performance",
		Help: "Disk performance metrics (read/write bytes and IOPS)",
	}, []string{"device", "activity"})
	networkActivity = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "game_network",
		Help: "Network activity metrics (bps, pps)",
	}, []string{"interface", "activity", "metric"})
	netstatConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "game_netstat",
		Help: "Network connections by port and state",
	}, []string{"port", "state"})
)

func init() {
	// Register metrics with Prometheus
	prometheus.MustRegister(serverUptime)
	prometheus.MustRegister(systemLoad)
	prometheus.MustRegister(cpuUsage)
	prometheus.MustRegister(memoryUsagePercent)
	prometheus.MustRegister(memoryTotalSize)
	prometheus.MustRegister(memoryUsageBytes)
	prometheus.MustRegister(memoryFreeBytes)
	prometheus.MustRegister(memoryFreePercent)
	prometheus.MustRegister(diskUsagePercent)
	prometheus.MustRegister(diskSize)
	prometheus.MustRegister(diskUsed)
	prometheus.MustRegister(diskAvailable)
	prometheus.MustRegister(diskTotalSize)
	prometheus.MustRegister(diskTotalAvailableBytes)
	prometheus.MustRegister(diskTotalAvailablePercent)
	prometheus.MustRegister(diskTotalUsedBytes)
	prometheus.MustRegister(diskTotalUsedPercent)
	prometheus.MustRegister(diskPerformance)
	prometheus.MustRegister(networkActivity)
	prometheus.MustRegister(netstatConnections)
}

// Collect server uptime
func getUptime() float64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		log.Println("Error reading /proc/uptime:", err)
		return 0
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return 0
	}
	uptime, _ := strconv.ParseFloat(parts[0], 64)
	return uptime
}

func getSystemLoad() map[string]float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		log.Println("Error reading /proc/loadavg:", err)
		return nil
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return nil
	}

	load1m, _ := strconv.ParseFloat(fields[0], 64)
	load5m, _ := strconv.ParseFloat(fields[1], 64)
	load15m, _ := strconv.ParseFloat(fields[2], 64)

	return map[string]float64{
		"1m":  load1m,
		"5m":  load5m,
		"15m": load15m,
	}
}

// Collect CPU usage
func getCPUUsage() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		log.Println("Error reading /proc/stat:", err)
		return 0
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return 0
			}
			user, _ := strconv.ParseFloat(fields[1], 64)
			nice, _ := strconv.ParseFloat(fields[2], 64)
			system, _ := strconv.ParseFloat(fields[3], 64)
			idle, _ := strconv.ParseFloat(fields[4], 64)
			total := user + nice + system + idle
			busy := total - idle
			return (busy / total) * 100
		}
	}
	return 0
}

// Collect memory usage
func getMemoryUsage() (float64, float64, float64, float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		log.Println("Error reading /proc/meminfo:", err)
		return 0, 0, 0, 0
	}
	var memTotal, memFree, memAvailable float64
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			memTotal, _ = strconv.ParseFloat(parts[1], 64)
			memTotal *= 1024 // Convert KB to bytes
		} else if strings.HasPrefix(line, "MemFree:") {
			parts := strings.Fields(line)
			memFree, _ = strconv.ParseFloat(parts[1], 64)
			memFree *= 1024 // Convert KB to bytes
		} else if strings.HasPrefix(line, "MemAvailable:") {
			parts := strings.Fields(line)
			memAvailable, _ = strconv.ParseFloat(parts[1], 64)
			memAvailable *= 1024 // Convert KB to bytes
		}
	}
	memUsed := memTotal - memFree
	return (memUsed / memTotal) * 100, memTotal, memUsed, (memFree / memTotal) * 100
}

// Collect disk usage
func getDiskUsage() (map[string]map[string]float64, float64, float64, float64, float64, float64) {
	cmd := exec.Command("df", "-k") // Use -k to get sizes in KB
	out, err := cmd.Output()
	if err != nil {
		log.Println("Error running df command:", err)
		return nil, 0, 0, 0, 0, 0
	}

	diskMetrics := make(map[string]map[string]float64)
	var totalSize, totalUsed, totalAvailable float64
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if i == 0 { // Skip header
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		partition := fields[5]
		// Exclude partitions like /dev, /run, and /sys
		if strings.HasPrefix(partition, "/dev") || strings.HasPrefix(partition, "/run") || strings.HasPrefix(partition, "/sys") {
			continue
		}
		sizeKB, _ := strconv.ParseFloat(fields[1], 64)
		usedKB, _ := strconv.ParseFloat(fields[2], 64)
		availableKB, _ := strconv.ParseFloat(fields[3], 64)
		usePercent, _ := strconv.ParseFloat(strings.TrimSuffix(fields[4], "%"), 64)

		diskMetrics[partition] = map[string]float64{
			"size":        sizeKB * 1024,      // Convert KB to bytes
			"used":        usedKB * 1024,      // Convert KB to bytes
			"available":   availableKB * 1024, // Convert KB to bytes
			"use_percent": usePercent,
		}
		totalSize += sizeKB * 1024
		totalUsed += usedKB * 1024
		totalAvailable += availableKB * 1024
	}

	totalAvailablePercent := (totalAvailable / totalSize) * 100
	totalUsedPercent := (totalUsed / totalSize) * 100
	return diskMetrics, totalSize, totalUsed, totalAvailable, totalAvailablePercent, totalUsedPercent
}

func getDiskPerformance() map[string]map[string]float64 {
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		log.Println("Error reading /proc/diskstats:", err)
		return nil
	}

	diskMetrics := make(map[string]map[string]float64)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}

		device := fields[2]
		if strings.HasPrefix(device, "loop") || strings.HasPrefix(device, "ram") {
			continue // Skip loopback and RAM devices
		}

		// Parse read/write metrics
		readOps, _ := strconv.ParseFloat(fields[3], 64)
		readBytes, _ := strconv.ParseFloat(fields[5], 64)
		writeOps, _ := strconv.ParseFloat(fields[7], 64)
		writeBytes, _ := strconv.ParseFloat(fields[9], 64)

		// Convert sectors to bytes (assuming 512-byte sectors)
		readBytes *= 512
		writeBytes *= 512

		diskMetrics[device] = map[string]float64{
			"readbytes":  readBytes,
			"readiops":   readOps,
			"writebytes": writeBytes,
			"writeiops":  writeOps,
		}
	}
	return diskMetrics
}

// Collect network I/O
func getNetworkIO() map[string]map[string]float64 {
	cmd := exec.Command("cat", "/proc/net/dev")
	out, err := cmd.Output()
	if err != nil {
		log.Println("Error reading /proc/net/dev:", err)
		return nil
	}

	networkMetrics := make(map[string]map[string]float64)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") && !strings.Contains(line, "lo:") {
			fields := strings.Fields(line)
			interfaceName := strings.TrimSuffix(fields[0], ":")
			rxBytes, _ := strconv.ParseFloat(fields[1], 64)
			txBytes, _ := strconv.ParseFloat(fields[9], 64)
			rxPackets, _ := strconv.ParseFloat(fields[2], 64)
			txPackets, _ := strconv.ParseFloat(fields[10], 64)

			networkMetrics[interfaceName] = map[string]float64{
				"rx_bytes":   rxBytes * 8, // Convert bytes to bits
				"tx_bytes":   txBytes * 8, // Convert bytes to bits
				"rx_packets": rxPackets,
				"tx_packets": txPackets,
			}
		}
	}
	return networkMetrics
}

// Collect listening ports and established connections
func getNetstat() map[string]map[string]int {
	cmd := exec.Command("netstat", "-nat")
	out, err := cmd.Output()
	if err != nil {
		log.Println("Error running netstat command:", err)
		return nil
	}

	listeningPorts := make(map[string]bool)
	connectionStates := make(map[string]map[string]int)

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// Extract local address and port
		localAddr := fields[3]
		parts := strings.Split(localAddr, ":")
		if len(parts) < 2 {
			continue
		}
		port := parts[len(parts)-1]

		// Track listening ports
		if strings.Contains(line, "LISTEN") {
			listeningPorts[port] = true
		}

		// Check if the port is in listeningPorts
		if listeningPorts[port] {
			// Extract connection state
			state := fields[5]
			if state == "" {
				continue
			}

			// Initialize map for the port if not already present
			if _, exists := connectionStates[port]; !exists {
				connectionStates[port] = make(map[string]int)
			}

			// Increment the count for the specific state
			connectionStates[port][state]++
		}
	}

	return connectionStates
}

// Collect metrics periodically
func collectMetrics() {
	for {
		serverUptime.Set(getUptime())
		cpuUsage.Set(getCPUUsage())

		// System load metrics
		systemLoadMetrics := getSystemLoad()
		for duration, load := range systemLoadMetrics {
			systemLoad.WithLabelValues(duration).Set(load)
		}

		// Memory metrics
		memUsagePercent, memTotal, memUsed, memFreePercent := getMemoryUsage()
		memoryUsagePercent.Set(memUsagePercent)
		memoryTotalSize.Set(memTotal)
		memoryUsageBytes.Set(memUsed)
		memoryFreeBytes.Set(memTotal - memUsed)
		memoryFreePercent.Set(memFreePercent)

		// Disk Usage metrics
		diskMetrics, totalSize, totalUsed, totalUsedPercent, totalAvailable, totalAvailablePercent := getDiskUsage()
		diskTotalSize.Set(totalSize)
		diskTotalUsedBytes.Set(totalUsed)
		diskTotalUsedPercent.Set(totalUsedPercent)
		diskTotalAvailableBytes.Set(totalAvailable)
		diskTotalAvailablePercent.Set(totalAvailablePercent)

		for partition, metrics := range diskMetrics {
			diskUsagePercent.WithLabelValues(partition).Set(metrics["use_percent"])
			diskSize.WithLabelValues(partition).Set(metrics["size"])
			diskUsed.WithLabelValues(partition).Set(metrics["used"])
			diskAvailable.WithLabelValues(partition).Set(metrics["available"])
		}

		// Disk performance metrics
		diskPerformanceMetrics := getDiskPerformance()
		for device, metrics := range diskPerformanceMetrics {
			diskPerformance.WithLabelValues(device, "readbytes").Set(metrics["readbytes"])
			diskPerformance.WithLabelValues(device, "readiops").Set(metrics["readiops"])
			diskPerformance.WithLabelValues(device, "writebytes").Set(metrics["writebytes"])
			diskPerformance.WithLabelValues(device, "writeiops").Set(metrics["writeiops"])
		}

		// Network metrics
		networkMetrics := getNetworkIO()
		for iface, metrics := range networkMetrics {
			networkActivity.WithLabelValues(iface, "in", "bps").Set(metrics["rx_bytes"])
			networkActivity.WithLabelValues(iface, "out", "bps").Set(metrics["tx_bytes"])
			networkActivity.WithLabelValues(iface, "in", "pps").Set(metrics["rx_packets"])
			networkActivity.WithLabelValues(iface, "out", "pps").Set(metrics["tx_packets"])
		}

		// Netstat metrics
		connectionStates := getNetstat()

		// Reset all netstat metrics before updating
		netstatConnections.Reset()

		// Update netstat metrics for each port and state
		for port, states := range connectionStates {
			for state, count := range states {
				netstatConnections.WithLabelValues(port, state).Set(float64(count))
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func main() {
	// Start collecting metrics in the background
	go collectMetrics()

	// Serve metrics on /metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Game server exporter started on :9108")
	log.Fatal(http.ListenAndServe(":9108", nil))
}
