package soak

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"time"
)

// RunSoakEval builds all dependencies from config, runs the evaluation, and writes reports.
func RunSoakEval(ctx context.Context, config SoakConfig) (*SoakReport, error) {
	playerAdapter, err := BuildModelAdapter(config.PlayerModel)
	if err != nil {
		return nil, err
	}
	judgeAdapter, err := BuildModelAdapter(config.JudgeModel)
	if err != nil {
		return nil, err
	}

	httpClient := NewGameHTTPClient(config.BaseURL, config.UserToken, nil)
	runner := NewRunner(
		config,
		NewPlayerSimulator(playerAdapter),
		httpClient,
		NewJudge(judgeAdapter),
		WithRoundReporter(BuildCheckpointReporter(config.OutputPath)),
	)

	runCtx := ctx
	cancel := func() {}
	if config.TimeoutSeconds > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
	}
	defer cancel()

	report, err := runner.Run(runCtx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.OutputPath) != "" {
		if err := WriteReport(config.OutputPath, *report); err != nil {
			return nil, err
		}
		if err := WriteMarkdownReport(markdownReportPath(config.OutputPath), *report); err != nil {
			return nil, err
		}
	}
	return report, nil
}

// BuildCheckpointReporter returns a reporter that logs progress and writes partial reports every round.
func BuildCheckpointReporter(outputPath string) RoundReporter {
	return func(record RoundRecord, report SoakReport) {
		log.Printf(
			"soak eval round completed: round=%d success=%t score=%.2f latency_ms=%d success_rate=%.2f failures=%v",
			record.Round,
			record.Success,
			record.Score,
			record.LatencyMS,
			report.SuccessRate,
			record.FailureReasons,
		)
		if strings.TrimSpace(outputPath) == "" {
			return
		}
		if err := WriteReport(outputPath, report); err != nil {
			log.Printf("soak eval checkpoint JSON write failed: round=%d error=%v", record.Round, err)
		}
		if err := WriteMarkdownReport(markdownReportPath(outputPath), report); err != nil {
			log.Printf("soak eval checkpoint Markdown write failed: round=%d error=%v", record.Round, err)
		}
	}
}

func markdownReportPath(jsonPath string) string {
	extension := filepath.Ext(jsonPath)
	if extension == "" {
		return jsonPath + ".md"
	}
	return strings.TrimSuffix(jsonPath, extension) + ".md"
}
