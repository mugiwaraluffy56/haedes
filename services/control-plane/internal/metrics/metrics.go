package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Registry is a small, dependency-free metrics registry for the single
// control-plane process. It emits Prometheus text exposition and deliberately
// accepts only bounded, low-cardinality labels from callers.
type Registry struct {
	mu         sync.RWMutex
	counters   map[string]map[string]float64
	histograms map[string]map[string]histogram
}

type histogram struct {
	count uint64
	sum   float64
}

func NewRegistry() *Registry {
	return &Registry{
		counters:   make(map[string]map[string]float64),
		histograms: make(map[string]map[string]histogram),
	}
}

func (registry *Registry) Inc(name string, labels map[string]string) {
	registry.Add(name, 1, labels)
}

func (registry *Registry) Add(name string, value float64, labels map[string]string) {
	if registry == nil || value == 0 {
		return
	}
	key := labelKey(labels)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.counters[name] == nil {
		registry.counters[name] = make(map[string]float64)
	}
	registry.counters[name][key] += value
}

func (registry *Registry) Observe(name string, value float64, labels map[string]string) {
	if registry == nil {
		return
	}
	key := labelKey(labels)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.histograms[name] == nil {
		registry.histograms[name] = make(map[string]histogram)
	}
	valueSet := registry.histograms[name][key]
	valueSet.count++
	valueSet.sum += value
	registry.histograms[name][key] = valueSet
}

func (registry *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = writer.Write([]byte(registry.Exposition()))
	})
}

func (registry *Registry) Exposition() string {
	if registry == nil {
		return ""
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	var builder strings.Builder

	counterNames := sortedCounterNames(registry.counters)
	for _, name := range counterNames {
		fmt.Fprintf(&builder, "# TYPE %s counter\n", name)
		for _, key := range sortedLabelKeys(registry.counters[name]) {
			fmt.Fprintf(&builder, "%s%s %s\n", name, key, strconv.FormatFloat(registry.counters[name][key], 'f', -1, 64))
		}
	}
	histogramNames := make([]string, 0, len(registry.histograms))
	for name := range registry.histograms {
		histogramNames = append(histogramNames, name)
	}
	sort.Strings(histogramNames)
	for _, name := range histogramNames {
		fmt.Fprintf(&builder, "# TYPE %s histogram\n", name)
		for _, key := range sortedHistogramKeys(registry.histograms[name]) {
			value := registry.histograms[name][key]
			fmt.Fprintf(&builder, "%s_count%s %d\n", name, key, value.count)
			fmt.Fprintf(&builder, "%s_sum%s %s\n", name, key, strconv.FormatFloat(value.sum, 'f', -1, 64))
		}
	}
	return builder.String()
}

func labelKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+`="`+strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(labels[key])+`"`)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func sortedCounterNames(values map[string]map[string]float64) []string {
	result := make([]string, 0, len(values))
	for name := range values {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func sortedLabelKeys(values map[string]float64) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func sortedHistogramKeys(values map[string]histogram) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
