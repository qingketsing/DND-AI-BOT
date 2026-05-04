package rag

func BuildQueryMetrics(query Query, gold GoldsetEntry, retrievedChunkIDs []string, backend string, ks []int) QueryEvalRecord {
	record := QueryEvalRecord{
		QueryID:           query.ID,
		KnowledgeBase:     query.KnowledgeBase,
		QueryType:         query.QueryType,
		Backend:           backend,
		RetrievedChunkIDs: append([]string(nil), retrievedChunkIDs...),
		RelevantChunkIDs:  append([]string(nil), gold.RelevantChunkIDs...),
		RecallAtK:         make(map[int]float64, len(ks)),
	}

	relevant := make(map[string]struct{}, len(gold.RelevantChunkIDs))
	for _, chunkID := range gold.RelevantChunkIDs {
		relevant[chunkID] = struct{}{}
	}

	for index, chunkID := range retrievedChunkIDs {
		if _, ok := relevant[chunkID]; ok {
			record.FirstRelevantRank = index + 1
			record.MRR = 1.0 / float64(record.FirstRelevantRank)
			break
		}
	}

	totalRelevant := float64(len(relevant))
	for _, k := range ks {
		if k <= 0 || totalRelevant == 0 {
			record.RecallAtK[k] = 0
			continue
		}
		limit := minInt(k, len(retrievedChunkIDs))
		hits := 0
		seen := make(map[string]struct{}, limit)
		for _, chunkID := range retrievedChunkIDs[:limit] {
			if _, ok := seen[chunkID]; ok {
				continue
			}
			seen[chunkID] = struct{}{}
			if _, ok := relevant[chunkID]; ok {
				hits++
			}
		}
		record.RecallAtK[k] = float64(hits) / totalRelevant
	}

	return record
}

func BuildMetricsReport(ks []int, records []QueryEvalRecord) MetricsReport {
	report := MetricsReport{
		Overall:         summarizeByBackend(ks, records),
		ByKnowledgeBase: make(map[string]map[string]MetricSummary),
		ByQueryType:     make(map[string]map[string]MetricSummary),
		Records:         append([]QueryEvalRecord(nil), records...),
	}

	byKnowledgeBase := make(map[string][]QueryEvalRecord)
	byQueryType := make(map[string][]QueryEvalRecord)
	for _, record := range records {
		byKnowledgeBase[record.KnowledgeBase] = append(byKnowledgeBase[record.KnowledgeBase], record)
		byQueryType[record.QueryType] = append(byQueryType[record.QueryType], record)
	}

	for key, groupedRecords := range byKnowledgeBase {
		report.ByKnowledgeBase[key] = summarizeByBackend(ks, groupedRecords)
	}
	for key, groupedRecords := range byQueryType {
		report.ByQueryType[key] = summarizeByBackend(ks, groupedRecords)
	}

	return report
}

func summarizeByBackend(ks []int, records []QueryEvalRecord) map[string]MetricSummary {
	grouped := make(map[string][]QueryEvalRecord)
	for _, record := range records {
		grouped[record.Backend] = append(grouped[record.Backend], record)
	}

	summary := make(map[string]MetricSummary, len(grouped))
	for backend, backendRecords := range grouped {
		metric := MetricSummary{
			QueryCount: len(backendRecords),
			RecallAtK:  make(map[int]float64, len(ks)),
		}
		if len(backendRecords) == 0 {
			summary[backend] = metric
			continue
		}
		for _, record := range backendRecords {
			metric.MRR += record.MRR
			for _, k := range ks {
				metric.RecallAtK[k] += record.RecallAtK[k]
			}
		}
		denominator := float64(len(backendRecords))
		metric.MRR /= denominator
		for _, k := range ks {
			metric.RecallAtK[k] /= denominator
		}
		summary[backend] = metric
	}
	return summary
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
