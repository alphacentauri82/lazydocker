package commands

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jesseduffield/asciigraph"
	"github.com/jesseduffield/lazydocker/pkg/config"
	"github.com/jesseduffield/lazydocker/pkg/utils"
	"github.com/mcuadros/go-lookup"
)

// RecordedStats contains both the container stats we've received from docker, and our own derived stats  from those container stats. When configuring a graph, you're basically specifying the path of a value in this struct
type RecordedStats struct {
	ClientStats  ContainerStats
	DerivedStats DerivedStats
	RecordedAt   time.Time
}

// DerivedStats contains some useful stats that we've calculated based on the raw container stats that we got back from docker
type DerivedStats struct {
	CPUPercentage    float64
	MemoryPercentage float64
}

// ContainerStats autogenerated at https://mholt.github.io/json-to-go/
type ContainerStats struct {
	Read      time.Time `json:"read"`
	Preread   time.Time `json:"preread"`
	PidsStats struct {
		Current int `json:"current"`
	} `json:"pids_stats"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int    `json:"value"`
		} `json:"io_service_bytes_recursive"`
		IoServicedRecursive []struct {
			Major int    `json:"major"`
			Minor int    `json:"minor"`
			Op    string `json:"op"`
			Value int    `json:"value"`
		} `json:"io_serviced_recursive"`
		IoQueueRecursive       []interface{} `json:"io_queue_recursive"`
		IoServiceTimeRecursive []interface{} `json:"io_service_time_recursive"`
		IoWaitTimeRecursive    []interface{} `json:"io_wait_time_recursive"`
		IoMergedRecursive      []interface{} `json:"io_merged_recursive"`
		IoTimeRecursive        []interface{} `json:"io_time_recursive"`
		SectorsRecursive       []interface{} `json:"sectors_recursive"`
	} `json:"blkio_stats"`
	NumProcs     int `json:"num_procs"`
	StorageStats struct {
	} `json:"storage_stats"`
	CPUStats struct {
		CPUUsage struct {
			TotalUsage        int64   `json:"total_usage"`
			PercpuUsage       []int64 `json:"percpu_usage"`
			UsageInKernelmode int64   `json:"usage_in_kernelmode"`
			UsageInUsermode   int64   `json:"usage_in_usermode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		OnlineCpus     int   `json:"online_cpus"`
		ThrottlingData struct {
			Periods          int `json:"periods"`
			ThrottledPeriods int `json:"throttled_periods"`
			ThrottledTime    int `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"cpu_stats"`
	PrecpuStats struct {
		CPUUsage struct {
			TotalUsage        int64   `json:"total_usage"`
			PercpuUsage       []int64 `json:"percpu_usage"`
			UsageInKernelmode int64   `json:"usage_in_kernelmode"`
			UsageInUsermode   int64   `json:"usage_in_usermode"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		OnlineCpus     int   `json:"online_cpus"`
		ThrottlingData struct {
			Periods          int `json:"periods"`
			ThrottledPeriods int `json:"throttled_periods"`
			ThrottledTime    int `json:"throttled_time"`
		} `json:"throttling_data"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage    int `json:"usage"`
		MaxUsage int `json:"max_usage"`
		Stats    struct {
			ActiveAnon              int   `json:"active_anon"`
			ActiveFile              int   `json:"active_file"`
			Cache                   int   `json:"cache"`
			Dirty                   int   `json:"dirty"`
			HierarchicalMemoryLimit int64 `json:"hierarchical_memory_limit"`
			HierarchicalMemswLimit  int64 `json:"hierarchical_memsw_limit"`
			InactiveAnon            int   `json:"inactive_anon"`
			InactiveFile            int   `json:"inactive_file"`
			MappedFile              int   `json:"mapped_file"`
			Pgfault                 int   `json:"pgfault"`
			Pgmajfault              int   `json:"pgmajfault"`
			Pgpgin                  int   `json:"pgpgin"`
			Pgpgout                 int   `json:"pgpgout"`
			Rss                     int   `json:"rss"`
			RssHuge                 int   `json:"rss_huge"`
			TotalActiveAnon         int   `json:"total_active_anon"`
			TotalActiveFile         int   `json:"total_active_file"`
			TotalCache              int   `json:"total_cache"`
			TotalDirty              int   `json:"total_dirty"`
			TotalInactiveAnon       int   `json:"total_inactive_anon"`
			TotalInactiveFile       int   `json:"total_inactive_file"`
			TotalMappedFile         int   `json:"total_mapped_file"`
			TotalPgfault            int   `json:"total_pgfault"`
			TotalPgmajfault         int   `json:"total_pgmajfault"`
			TotalPgpgin             int   `json:"total_pgpgin"`
			TotalPgpgout            int   `json:"total_pgpgout"`
			TotalRss                int   `json:"total_rss"`
			TotalRssHuge            int   `json:"total_rss_huge"`
			TotalUnevictable        int   `json:"total_unevictable"`
			TotalWriteback          int   `json:"total_writeback"`
			Unevictable             int   `json:"unevictable"`
			Writeback               int   `json:"writeback"`
		} `json:"stats"`
		Limit int64 `json:"limit"`
	} `json:"memory_stats"`
	Name     string `json:"name"`
	ID       string `json:"id"`
	Networks struct {
		Eth0 struct {
			RxBytes   int `json:"rx_bytes"`
			RxPackets int `json:"rx_packets"`
			RxErrors  int `json:"rx_errors"`
			RxDropped int `json:"rx_dropped"`
			TxBytes   int `json:"tx_bytes"`
			TxPackets int `json:"tx_packets"`
			TxErrors  int `json:"tx_errors"`
			TxDropped int `json:"tx_dropped"`
		} `json:"eth0"`
	} `json:"networks"`
}

// CalculateContainerCPUPercentage calculates the cpu usage of the container as a percent of total CPU usage
// to calculate CPU usage, we take the increase in CPU time from the container since the last poll, divide that by the total increase in CPU time since the last poll, times by the number of cores, and times by 100 to get a percentage
// I'm not entirely sure why we need to multiply by the number of cores, but the numbers work
func (s *ContainerStats) CalculateContainerCPUPercentage() float64 {
	cpuUsageDelta := s.CPUStats.CPUUsage.TotalUsage - s.PrecpuStats.CPUUsage.TotalUsage
	cpuTotalUsageDelta := s.CPUStats.SystemCPUUsage - s.PrecpuStats.SystemCPUUsage
	numberOfCores := len(s.CPUStats.CPUUsage.PercpuUsage)

	value := float64(cpuUsageDelta*100) * float64(numberOfCores) / float64(cpuTotalUsageDelta)
	if math.IsNaN(value) {
		return 0
	}
	return value
}

// CalculateContainerMemoryUsage calculates the memory usage of the container as a percent of total available memory
func (s *ContainerStats) CalculateContainerMemoryUsage() float64 {

	value := float64(s.MemoryStats.Usage*100) / float64(s.MemoryStats.Limit)
	if math.IsNaN(value) {
		return 0
	}
	return value
}

// RenderStats returns a string containing the rendered stats of the container
func (c *Container) RenderStats(viewWidth int) (string, error) {
	history := c.StatHistory
	if len(history) == 0 {
		return "", nil
	}
	currentStats := history[len(history)-1]

	graphSpecs := c.Config.UserConfig.Stats.Graphs
	graphs := make([]string, len(graphSpecs))
	for i, spec := range graphSpecs {
		graph, err := c.PlotGraph(spec, viewWidth-10)
		if err != nil {
			return "", err
		}
		graphs[i] = utils.ColoredString(graph, utils.GetColorAttribute(spec.Color))
	}

	pidsCount := fmt.Sprintf("PIDs: %d", currentStats.ClientStats.PidsStats.Current)
	dataReceived := fmt.Sprintf("Traffic received: %s", utils.FormatDecimalBytes(currentStats.ClientStats.Networks.Eth0.RxBytes))
	dataSent := fmt.Sprintf("Traffic sent: %s", utils.FormatDecimalBytes(currentStats.ClientStats.Networks.Eth0.TxBytes))

	originalJSON, err := json.MarshalIndent(currentStats, "", "  ")
	if err != nil {
		return "", err
	}

	contents := fmt.Sprintf("\n\n%s\n\n%s\n\n%s\n%s\n\n%s",
		utils.ColoredString(strings.Join(graphs, "\n\n"), color.FgGreen),
		pidsCount,
		dataReceived,
		dataSent,
		string(originalJSON),
	)

	return contents, nil
}

// PlotGraph returns the plotted graph based on the graph spec and the stat history
func (c *Container) PlotGraph(spec config.GraphConfig, width int) (string, error) {
	data := make([]float64, len(c.StatHistory))

	max := spec.Max
	min := spec.Min
	for i, stats := range c.StatHistory {
		value, err := lookup.LookupString(stats, spec.StatPath)
		if err != nil {
			return "Could not find key: " + spec.StatPath, nil
		}
		floatValue, err := getFloat(value.Interface())
		if err != nil {
			return "", err
		}
		if spec.MinType == "" {
			if i == 0 {
				min = floatValue
			} else {
				if floatValue < min {
					min = floatValue
				}
			}
		}

		if spec.MaxType == "" {
			if i == 0 {
				max = floatValue
			} else {
				if floatValue > max {
					max = floatValue
				}
			}
		}

		data[i] = floatValue
	}

	height := 10
	if spec.Height > 0 {
		height = spec.Height
	}

	return asciigraph.Plot(
		data,
		asciigraph.Height(height),
		asciigraph.Width(width),
		asciigraph.Min(min),
		asciigraph.Max(max),
		asciigraph.Caption(fmt.Sprintf("%s: %0.2f (%v)", spec.Caption, data[len(data)-1], time.Since(c.StatHistory[0].RecordedAt))),
	), nil
}

// from Dave C's answer at https://stackoverflow.com/questions/20767724/converting-unknown-interface-to-float64-in-golang
func getFloat(unk interface{}) (float64, error) {
	floatType := reflect.TypeOf(float64(0))
	stringType := reflect.TypeOf("")

	switch i := unk.(type) {
	case float64:
		return i, nil
	case float32:
		return float64(i), nil
	case int64:
		return float64(i), nil
	case int32:
		return float64(i), nil
	case int:
		return float64(i), nil
	case uint64:
		return float64(i), nil
	case uint32:
		return float64(i), nil
	case uint:
		return float64(i), nil
	case string:
		return strconv.ParseFloat(i, 64)
	default:
		v := reflect.ValueOf(unk)
		v = reflect.Indirect(v)
		if v.Type().ConvertibleTo(floatType) {
			fv := v.Convert(floatType)
			return fv.Float(), nil
		} else if v.Type().ConvertibleTo(stringType) {
			sv := v.Convert(stringType)
			s := sv.String()
			return strconv.ParseFloat(s, 64)
		} else {
			return math.NaN(), fmt.Errorf("Can't convert %v to float64", v.Type())
		}
	}
}
