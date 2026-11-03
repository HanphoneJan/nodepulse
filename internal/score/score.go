package score

import (
	"math"
	"sort"
)

// Result is the outcome of testing one node.
type Result struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Success  int     `json:"success"`
	Total    int     `json:"total"`
	MinMs    int     `json:"min_ms"`
	MedianMs int     `json:"median_ms"`
	P90Ms    int     `json:"p90_ms"`
	MaxMs    int     `json:"max_ms"`
	JitterMs int     `json:"jitter_ms"`
	Score    float64 `json:"score"`

	// Latencies is optional and can be omitted from CLI output with --no-raw.
	Latencies []int `json:"latencies,omitempty"`
	// LastError is the last transport/controller error seen for this node.
	LastError string `json:"last_error,omitempty"`
	// Source is a human-readable subscription/profile label for merged tests.
	Source string `json:"source,omitempty"`
}

// SuccessRate returns success / total as a fraction.
func (r Result) SuccessRate() float64 {
	if r.Total <= 0 {
		return 0
	}
	return float64(r.Success) / float64(r.Total)
}

// FromLatencies builds a Result from successful latency samples.
func FromLatencies(name, typ string, latencies []int, total int) Result {
	r := Result{
		Name:      name,
		Type:      typ,
		Success:   len(latencies),
		Total:     total,
		Latencies: append([]int(nil), latencies...),
	}
	if len(latencies) > 0 {
		sorted := append([]int(nil), latencies...)
		sort.Ints(sorted)
		r.MinMs = sorted[0]
		r.MaxMs = sorted[len(sorted)-1]
		r.MedianMs = percentile(sorted, 50)
		r.P90Ms = percentile(sorted, 90)
		r.JitterMs = max(0, r.P90Ms-r.MedianMs)
	}
	r.Score = Compute(r)
	return r
}

// Compute applies the scoring formula:
//
//	successRate   = success / total
//	latencyScore  = clamp(1 - (p90-100)/900, 0, 1)
//	stabilityScore= clamp(1 - jitter/500, 0, 1)
//	score         = 0.5*successRate + 0.3*latencyScore + 0.2*stabilityScore
//
// Nodes below 80% success are additionally scaled by their success rate so
// that "sometimes works" nodes cannot outrank reliable ones.
func Compute(r Result) float64 {
	if r.Total <= 0 || r.Success <= 0 {
		return 0
	}
	sr := r.SuccessRate()
	latency := clamp(1-(float64(r.P90Ms)-100)/900, 0, 1)
	stability := clamp(1-float64(r.JitterMs)/500, 0, 1)
	score := 0.5*sr + 0.3*latency + 0.2*stability
	if sr < 0.8 {
		score *= sr
	}
	return math.Round(score*10000) / 10000
}

// Rank sorts results by success rate, score, P90 and jitter. Input order is
// preserved for exact ties.
func Rank(results []Result) []Result {
	out := append([]Result(nil), results...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if sa, sb := a.SuccessRate(), b.SuccessRate(); sa != sb {
			return sa > sb
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.P90Ms != b.P90Ms {
			return a.P90Ms < b.P90Ms
		}
		return a.JitterMs < b.JitterMs
	})
	return out
}

func percentile(sorted []int, p float64) int {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
