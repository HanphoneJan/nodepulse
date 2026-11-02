package score

import "testing"

func TestFromLatencies(t *testing.T) {
	r := FromLatencies("A", "ss", []int{50, 55, 60, 58, 52}, 5)
	if r.Success != 5 || r.Total != 5 {
		t.Fatalf("success = %d/%d, want 5/5", r.Success, r.Total)
	}
	if r.MinMs != 50 || r.MaxMs != 60 {
		t.Fatalf("min/max = %d/%d, want 50/60", r.MinMs, r.MaxMs)
	}
	if r.MedianMs != 55 {
		t.Fatalf("median = %d, want 55", r.MedianMs)
	}
	if r.P90Ms != 60 {
		t.Fatalf("p90 = %d, want 60", r.P90Ms)
	}
	if r.JitterMs != 5 {
		t.Fatalf("jitter = %d, want 5", r.JitterMs)
	}
	if r.Score < 0.9 {
		t.Fatalf("score = %.3f, want >= 0.9", r.Score)
	}
}

func TestFailuresLowerScore(t *testing.T) {
	good := FromLatencies("good", "ss", []int{100, 100, 100, 100, 100, 100, 100, 100, 100, 100}, 10)
	bad := FromLatencies("bad", "ss", []int{100}, 10)
	if bad.Score >= good.Score {
		t.Fatalf("bad score %.3f should be lower than good %.3f", bad.Score, good.Score)
	}
	if bad.SuccessRate() != 0.1 {
		t.Fatalf("success rate = %.2f, want 0.1", bad.SuccessRate())
	}
}

func TestRankPrefersSuccessThenScore(t *testing.T) {
	results := []Result{
		FromLatencies("slow-but-stable", "ss", repeat(300, 10), 10),
		FromLatencies("fast-but-unstable", "ss", []int{100, 100, 100}, 10),
		FromLatencies("fast-and-stable", "ss", repeat(120, 10), 10),
	}
	ranked := Rank(results)
	if ranked[0].Name != "fast-and-stable" {
		t.Fatalf("first = %q, want fast-and-stable", ranked[0].Name)
	}
	if ranked[len(ranked)-1].Name != "fast-but-unstable" {
		t.Fatalf("last = %q, want fast-but-unstable", ranked[len(ranked)-1].Name)
	}
}

func repeat(v, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = v
	}
	return out
}
