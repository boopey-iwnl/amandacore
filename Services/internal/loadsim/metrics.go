package loadsim

import (
	"math"
	"sort"
	"time"
)

type TickMetrics struct {
	Count           int           `json:"count"`
	AverageDuration time.Duration `json:"averageDuration"`
	MaxDuration     time.Duration `json:"maxDuration"`
	P50             time.Duration `json:"p50"`
	P95             time.Duration `json:"p95"`
	P99             time.Duration `json:"p99"`
}

type QueueMetrics struct {
	MaxDepth     int     `json:"maxDepth"`
	AverageDepth float64 `json:"averageDepth"`
	Samples      int     `json:"samples"`
}

type CommandMetrics struct {
	Sent             int            `json:"sent"`
	Accepted         int            `json:"accepted"`
	Rejected         int            `json:"rejected"`
	RejectionReasons map[string]int `json:"rejectionReasons"`
}

type TransitionMetrics struct {
	Requested int            `json:"requested"`
	Completed int            `json:"completed"`
	Rejected  int            `json:"rejected"`
	ByRoute   map[string]int `json:"byRoute"`
}

type ReconnectMetrics struct {
	Attempts  int `json:"attempts"`
	Successes int `json:"successes"`
	Failures  int `json:"failures"`
}

func SummarizeTickDurations(values []time.Duration) TickMetrics {
	if len(values) == 0 {
		return TickMetrics{}
	}
	var total time.Duration
	var maxDuration time.Duration
	for _, value := range values {
		total += value
		if value > maxDuration {
			maxDuration = value
		}
	}
	return TickMetrics{
		Count:           len(values),
		AverageDuration: total / time.Duration(len(values)),
		MaxDuration:     maxDuration,
		P50:             PercentileDuration(values, 0.50),
		P95:             PercentileDuration(values, 0.95),
		P99:             PercentileDuration(values, 0.99),
	}
}

func PercentileDuration(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	if percentile <= 0 {
		percentile = 0
	}
	if percentile > 1 {
		percentile = 1
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(left int, right int) bool {
		return sorted[left] < sorted[right]
	})
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func addQueueSample(metrics *QueueMetrics, depth int) {
	if depth > metrics.MaxDepth {
		metrics.MaxDepth = depth
	}
	metrics.AverageDepth = ((metrics.AverageDepth * float64(metrics.Samples)) + float64(depth)) / float64(metrics.Samples+1)
	metrics.Samples++
}
