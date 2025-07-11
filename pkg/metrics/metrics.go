package metrics

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Metrics tracks various application metrics
type Metrics struct {
	mu           sync.RWMutex
	startTime    time.Time
	counters     map[string]int64
	gauges       map[string]float64
	histograms   map[string]*Histogram
	timers       map[string]*Timer
	enabled      bool
}

// Histogram tracks distribution of values
type Histogram struct {
	values []float64
	mu     sync.RWMutex
}

// Timer tracks timing information
type Timer struct {
	start    time.Time
	duration time.Duration
	active   bool
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		startTime:  time.Now(),
		counters:   make(map[string]int64),
		gauges:     make(map[string]float64),
		histograms: make(map[string]*Histogram),
		timers:     make(map[string]*Timer),
		enabled:    true,
	}
}

// Enable enables metrics collection
func (m *Metrics) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = true
}

// Disable disables metrics collection
func (m *Metrics) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
}

// IsEnabled returns true if metrics collection is enabled
func (m *Metrics) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// IncCounter increments a counter metric
func (m *Metrics) IncCounter(name string) {
	m.AddCounter(name, 1)
}

// AddCounter adds a value to a counter metric
func (m *Metrics) AddCounter(name string, value int64) {
	if !m.IsEnabled() {
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

// SetGauge sets a gauge metric
func (m *Metrics) SetGauge(name string, value float64) {
	if !m.IsEnabled() {
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

// AddToHistogram adds a value to a histogram
func (m *Metrics) AddToHistogram(name string, value float64) {
	if !m.IsEnabled() {
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.histograms[name]; !exists {
		m.histograms[name] = &Histogram{
			values: make([]float64, 0),
		}
	}
	
	m.histograms[name].mu.Lock()
	m.histograms[name].values = append(m.histograms[name].values, value)
	m.histograms[name].mu.Unlock()
}

// StartTimer starts a timer
func (m *Metrics) StartTimer(name string) *Timer {
	if !m.IsEnabled() {
		return &Timer{active: false}
	}
	
	timer := &Timer{
		start:  time.Now(),
		active: true,
	}
	
	m.mu.Lock()
	m.timers[name] = timer
	m.mu.Unlock()
	
	return timer
}

// StopTimer stops a timer and records the duration
func (m *Metrics) StopTimer(name string) time.Duration {
	if !m.IsEnabled() {
		return 0
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if timer, exists := m.timers[name]; exists && timer.active {
		timer.duration = time.Since(timer.start)
		timer.active = false
		return timer.duration
	}
	
	return 0
}

// GetCounter returns a counter value
func (m *Metrics) GetCounter(name string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[name]
}

// GetGauge returns a gauge value
func (m *Metrics) GetGauge(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gauges[name]
}

// GetHistogramStats returns histogram statistics
func (m *Metrics) GetHistogramStats(name string) *HistogramStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	histogram, exists := m.histograms[name]
	if !exists {
		return nil
	}
	
	histogram.mu.RLock()
	defer histogram.mu.RUnlock()
	
	if len(histogram.values) == 0 {
		return &HistogramStats{}
	}
	
	// Calculate statistics
	var sum float64
	min := histogram.values[0]
	max := histogram.values[0]
	
	for _, value := range histogram.values {
		sum += value
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	
	mean := sum / float64(len(histogram.values))
	
	return &HistogramStats{
		Count: len(histogram.values),
		Sum:   sum,
		Mean:  mean,
		Min:   min,
		Max:   max,
	}
}

// HistogramStats contains histogram statistics
type HistogramStats struct {
	Count int
	Sum   float64
	Mean  float64
	Min   float64
	Max   float64
}

// GetTimerDuration returns the duration of a timer
func (m *Metrics) GetTimerDuration(name string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if timer, exists := m.timers[name]; exists {
		return timer.duration
	}
	
	return 0
}

// GetAllMetrics returns all metrics
func (m *Metrics) GetAllMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	metrics := make(map[string]interface{})
	
	// Add counters
	for name, value := range m.counters {
		metrics[fmt.Sprintf("counter_%s", name)] = value
	}
	
	// Add gauges
	for name, value := range m.gauges {
		metrics[fmt.Sprintf("gauge_%s", name)] = value
	}
	
	// Add histogram stats
	for name := range m.histograms {
		stats := m.GetHistogramStats(name)
		if stats != nil {
			metrics[fmt.Sprintf("histogram_%s_count", name)] = stats.Count
			metrics[fmt.Sprintf("histogram_%s_sum", name)] = stats.Sum
			metrics[fmt.Sprintf("histogram_%s_mean", name)] = stats.Mean
			metrics[fmt.Sprintf("histogram_%s_min", name)] = stats.Min
			metrics[fmt.Sprintf("histogram_%s_max", name)] = stats.Max
		}
	}
	
	// Add timer durations
	for name, timer := range m.timers {
		if !timer.active {
			metrics[fmt.Sprintf("timer_%s_duration_ms", name)] = timer.duration.Milliseconds()
		}
	}
	
	return metrics
}

// GetSystemMetrics returns system metrics
func (m *Metrics) GetSystemMetrics() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return map[string]interface{}{
		"uptime_seconds":        time.Since(m.startTime).Seconds(),
		"memory_alloc_bytes":    memStats.Alloc,
		"memory_total_alloc_bytes": memStats.TotalAlloc,
		"memory_sys_bytes":      memStats.Sys,
		"memory_heap_alloc_bytes": memStats.HeapAlloc,
		"memory_heap_sys_bytes": memStats.HeapSys,
		"memory_heap_idle_bytes": memStats.HeapIdle,
		"memory_heap_inuse_bytes": memStats.HeapInuse,
		"memory_heap_objects":   memStats.HeapObjects,
		"gc_num":               memStats.NumGC,
		"gc_pause_total_ns":    memStats.PauseTotalNs,
		"goroutines":           runtime.NumGoroutine(),
	}
}

// RecordCommandExecution records command execution metrics
func (m *Metrics) RecordCommandExecution(command string, success bool, duration time.Duration) {
	m.IncCounter(fmt.Sprintf("command_total_%s", command))
	
	if success {
		m.IncCounter(fmt.Sprintf("command_success_%s", command))
	} else {
		m.IncCounter(fmt.Sprintf("command_error_%s", command))
	}
	
	m.AddToHistogram(fmt.Sprintf("command_duration_%s", command), float64(duration.Milliseconds()))
}

// RecordSongEvent records song-related events
func (m *Metrics) RecordSongEvent(event string) {
	m.IncCounter(fmt.Sprintf("song_event_%s", event))
}

// RecordQueueEvent records queue-related events
func (m *Metrics) RecordQueueEvent(event string, queueSize int) {
	m.IncCounter(fmt.Sprintf("queue_event_%s", event))
	m.SetGauge("queue_size", float64(queueSize))
}

// RecordCacheEvent records cache-related events
func (m *Metrics) RecordCacheEvent(event string, cacheSize int64) {
	m.IncCounter(fmt.Sprintf("cache_event_%s", event))
	m.SetGauge("cache_size_bytes", float64(cacheSize))
}

// RecordDiscordEvent records Discord-related events
func (m *Metrics) RecordDiscordEvent(event string) {
	m.IncCounter(fmt.Sprintf("discord_event_%s", event))
}

// RecordYouTubeEvent records YouTube-related events
func (m *Metrics) RecordYouTubeEvent(event string) {
	m.IncCounter(fmt.Sprintf("youtube_event_%s", event))
}

// RecordAudioEvent records audio-related events
func (m *Metrics) RecordAudioEvent(event string, duration time.Duration) {
	m.IncCounter(fmt.Sprintf("audio_event_%s", event))
	if duration > 0 {
		m.AddToHistogram(fmt.Sprintf("audio_duration_%s", event), float64(duration.Milliseconds()))
	}
}

// RecordError records error events
func (m *Metrics) RecordError(errorType string) {
	m.IncCounter(fmt.Sprintf("error_%s", errorType))
}

// RecordUserAction records user actions
func (m *Metrics) RecordUserAction(action string, userID string) {
	m.IncCounter(fmt.Sprintf("user_action_%s", action))
	m.IncCounter(fmt.Sprintf("user_total_%s", userID))
}

// RecordGuildAction records guild actions
func (m *Metrics) RecordGuildAction(action string, guildID string) {
	m.IncCounter(fmt.Sprintf("guild_action_%s", action))
	m.IncCounter(fmt.Sprintf("guild_total_%s", guildID))
}

// GetMetricsSummary returns a summary of key metrics
func (m *Metrics) GetMetricsSummary() MetricsSummary {
	allMetrics := m.GetAllMetrics()
	systemMetrics := m.GetSystemMetrics()
	
	// Combine all metrics
	combined := make(map[string]interface{})
	for k, v := range allMetrics {
		combined[k] = v
	}
	for k, v := range systemMetrics {
		combined[k] = v
	}
	
	return MetricsSummary{
		Timestamp: time.Now(),
		Uptime:    time.Since(m.startTime),
		Metrics:   combined,
	}
}

// MetricsSummary contains a summary of metrics
type MetricsSummary struct {
	Timestamp time.Time
	Uptime    time.Duration
	Metrics   map[string]interface{}
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.counters = make(map[string]int64)
	m.gauges = make(map[string]float64)
	m.histograms = make(map[string]*Histogram)
	m.timers = make(map[string]*Timer)
	m.startTime = time.Now()
}

// MonitoringCollector collects metrics for monitoring
type MonitoringCollector struct {
	metrics  *Metrics
	interval time.Duration
	stopCh   chan struct{}
}

// NewMonitoringCollector creates a new monitoring collector
func NewMonitoringCollector(metrics *Metrics, interval time.Duration) *MonitoringCollector {
	return &MonitoringCollector{
		metrics:  metrics,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start starts the monitoring collector
func (c *MonitoringCollector) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.collectSystemMetrics()
		}
	}
}

// Stop stops the monitoring collector
func (c *MonitoringCollector) Stop() {
	close(c.stopCh)
}

// collectSystemMetrics collects system metrics
func (c *MonitoringCollector) collectSystemMetrics() {
	// Update system metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	c.metrics.SetGauge("system_memory_alloc_mb", float64(memStats.Alloc)/1024/1024)
	c.metrics.SetGauge("system_memory_sys_mb", float64(memStats.Sys)/1024/1024)
	c.metrics.SetGauge("system_goroutines", float64(runtime.NumGoroutine()))
	c.metrics.SetGauge("system_gc_count", float64(memStats.NumGC))
}

// Global metrics instance
var globalMetrics *Metrics
var metricsOnce sync.Once

// GetGlobalMetrics returns the global metrics instance
func GetGlobalMetrics() *Metrics {
	metricsOnce.Do(func() {
		globalMetrics = NewMetrics()
	})
	return globalMetrics
}

// Package-level convenience functions
func IncCounter(name string) {
	GetGlobalMetrics().IncCounter(name)
}

func AddCounter(name string, value int64) {
	GetGlobalMetrics().AddCounter(name, value)
}

func SetGauge(name string, value float64) {
	GetGlobalMetrics().SetGauge(name, value)
}

func AddToHistogram(name string, value float64) {
	GetGlobalMetrics().AddToHistogram(name, value)
}

func StartTimer(name string) *Timer {
	return GetGlobalMetrics().StartTimer(name)
}

func StopTimer(name string) time.Duration {
	return GetGlobalMetrics().StopTimer(name)
}

func RecordCommandExecution(command string, success bool, duration time.Duration) {
	GetGlobalMetrics().RecordCommandExecution(command, success, duration)
}

func RecordSongEvent(event string) {
	GetGlobalMetrics().RecordSongEvent(event)
}

func RecordQueueEvent(event string, queueSize int) {
	GetGlobalMetrics().RecordQueueEvent(event, queueSize)
}

func RecordCacheEvent(event string, cacheSize int64) {
	GetGlobalMetrics().RecordCacheEvent(event, cacheSize)
}

func RecordDiscordEvent(event string) {
	GetGlobalMetrics().RecordDiscordEvent(event)
}

func RecordYouTubeEvent(event string) {
	GetGlobalMetrics().RecordYouTubeEvent(event)
}

func RecordAudioEvent(event string, duration time.Duration) {
	GetGlobalMetrics().RecordAudioEvent(event, duration)
}

func RecordError(errorType string) {
	GetGlobalMetrics().RecordError(errorType)
}

func RecordUserAction(action string, userID string) {
	GetGlobalMetrics().RecordUserAction(action, userID)
}

func RecordGuildAction(action string, guildID string) {
	GetGlobalMetrics().RecordGuildAction(action, guildID)
}

func GetMetricsSummary() MetricsSummary {
	return GetGlobalMetrics().GetMetricsSummary()
}