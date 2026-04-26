package loadsim

import (
	"testing"
	"time"
)

func TestPercentileDurationHandlesEmptySingleAndMultiValue(t *testing.T) {
	if got := PercentileDuration(nil, 0.95); got != 0 {
		t.Fatalf("empty percentile = %s", got)
	}
	if got := PercentileDuration([]time.Duration{12 * time.Millisecond}, 0.95); got != 12*time.Millisecond {
		t.Fatalf("single percentile = %s", got)
	}
	values := []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond, 40 * time.Millisecond}
	if got := PercentileDuration(values, 0.50); got != 20*time.Millisecond {
		t.Fatalf("p50 = %s", got)
	}
	if got := PercentileDuration(values, 0.95); got != 40*time.Millisecond {
		t.Fatalf("p95 = %s", got)
	}
}

func TestTickMetricsAverageAndMax(t *testing.T) {
	summary := SummarizeTickDurations([]time.Duration{10 * time.Millisecond, 30 * time.Millisecond})
	if summary.AverageDuration != 20*time.Millisecond {
		t.Fatalf("avg = %s", summary.AverageDuration)
	}
	if summary.MaxDuration != 30*time.Millisecond {
		t.Fatalf("max = %s", summary.MaxDuration)
	}
}
