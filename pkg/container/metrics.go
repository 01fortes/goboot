package container

import (
	"sync"
	"time"
)

// MetricsCollector collects and manages component metrics
type MetricsCollector interface {
	RecordDependencyCount(componentName string, count int)
	RecordInitDuration(componentName string, duration time.Duration)
	RecordStartDuration(componentName string, duration time.Duration)
	RecordStopDuration(componentName string, duration time.Duration)
	GetMetrics() map[string]*ComponentMetrics
}

// ComponentMetrics stores metrics for a component
type ComponentMetrics struct {
	Name            string
	InitDuration    time.Duration
	StartDuration   time.Duration
	StopDuration    time.Duration
	DependencyCount int
}

// defaultMetricsCollector implements MetricsCollector
type defaultMetricsCollector struct {
	metrics map[string]*ComponentMetrics
	mu      sync.RWMutex
	enabled bool
}

func newMetricsCollector(enabled bool) *defaultMetricsCollector {
	return &defaultMetricsCollector{
		metrics: make(map[string]*ComponentMetrics),
		enabled: enabled,
	}
}

func (c *defaultMetricsCollector) ensureMetricExists(componentName string) {
	if !c.enabled {
		return
	}

	if _, exists := c.metrics[componentName]; !exists {
		c.metrics[componentName] = &ComponentMetrics{
			Name: componentName,
		}
	}
}

func (c *defaultMetricsCollector) RecordDependencyCount(componentName string, count int) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureMetricExists(componentName)
	c.metrics[componentName].DependencyCount = count
}

func (c *defaultMetricsCollector) RecordInitDuration(componentName string, duration time.Duration) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureMetricExists(componentName)
	c.metrics[componentName].InitDuration = duration
}

func (c *defaultMetricsCollector) RecordStartDuration(componentName string, duration time.Duration) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureMetricExists(componentName)
	c.metrics[componentName].StartDuration = duration
}

func (c *defaultMetricsCollector) RecordStopDuration(componentName string, duration time.Duration) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureMetricExists(componentName)
	c.metrics[componentName].StopDuration = duration
}

func (c *defaultMetricsCollector) GetMetrics() map[string]*ComponentMetrics {
	if !c.enabled {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create a copy to avoid races
	result := make(map[string]*ComponentMetrics, len(c.metrics))
	for k, v := range c.metrics {
		copy := *v
		result[k] = &copy
	}

	return result
}
