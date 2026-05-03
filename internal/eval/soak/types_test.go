package soak

import "testing"

func TestBuildReportCalculatesSuccessRateAndLatency(t *testing.T) {
	report := BuildReport("session-1", []RoundRecord{
		{Round: 1, Success: true, LatencyMS: 100},
		{Round: 2, Success: false, LatencyMS: 300, FailureReasons: []string{"forgot_scene"}},
	})

	if report.SessionID != "session-1" {
		t.Fatalf("expected session id session-1, got %q", report.SessionID)
	}
	if report.Rounds != 2 {
		t.Fatalf("expected 2 rounds, got %d", report.Rounds)
	}
	if report.SuccessRounds != 1 {
		t.Fatalf("expected 1 successful round, got %d", report.SuccessRounds)
	}
	if report.SuccessRate != 0.5 {
		t.Fatalf("expected success rate 0.5, got %f", report.SuccessRate)
	}
	if report.AvgLatencyMS != 200 {
		t.Fatalf("expected avg latency 200, got %d", report.AvgLatencyMS)
	}
	if report.FailureReasonCounts["forgot_scene"] != 1 {
		t.Fatalf("expected forgot_scene count 1, got %d", report.FailureReasonCounts["forgot_scene"])
	}
}
