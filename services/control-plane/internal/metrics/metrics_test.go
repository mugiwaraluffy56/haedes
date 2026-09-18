package metrics

import (
	"strings"
	"sync"
	"testing"
)

func TestRegistryIsConcurrentAndEmitsStablePrometheusText(t *testing.T) {
	registry := NewRegistry()
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 25 {
				registry.Inc("command_total", map[string]string{"outcome": "success"})
				registry.Observe("command_duration_seconds", 0.25, map[string]string{"outcome": "success"})
			}
		}()
	}
	group.Wait()
	exposition := registry.Exposition()
	for _, expected := range []string{
		"# TYPE command_total counter",
		`command_total{outcome="success"} 200`,
		"# TYPE command_duration_seconds histogram",
		`command_duration_seconds_count{outcome="success"} 200`,
		`command_duration_seconds_sum{outcome="success"} 50`,
	} {
		if !strings.Contains(exposition, expected) {
			t.Fatalf("metrics exposition is missing %q:\n%s", expected, exposition)
		}
	}
}
