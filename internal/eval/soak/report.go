package soak

// BuildReport calculates aggregate metrics from round records.
func BuildReport(sessionID string, records []RoundRecord) SoakReport {
	report := SoakReport{
		SessionID:           sessionID,
		Rounds:              len(records),
		FailureReasonCounts: make(map[string]int),
		Records:             append([]RoundRecord(nil), records...),
	}
	var totalLatency int64
	for _, record := range records {
		if record.Success {
			report.SuccessRounds++
		}
		totalLatency += record.LatencyMS
		for _, reason := range record.FailureReasons {
			report.FailureReasonCounts[reason]++
		}
	}
	if len(records) > 0 {
		report.SuccessRate = float64(report.SuccessRounds) / float64(len(records))
		report.AvgLatencyMS = totalLatency / int64(len(records))
	}
	return report
}
