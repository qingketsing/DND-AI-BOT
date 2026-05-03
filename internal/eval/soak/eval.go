package soak

import (
	"context"
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
	runner := NewRunner(config, NewPlayerSimulator(playerAdapter), httpClient, NewJudge(judgeAdapter))

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

func markdownReportPath(jsonPath string) string {
	extension := filepath.Ext(jsonPath)
	if extension == "" {
		return jsonPath + ".md"
	}
	return strings.TrimSuffix(jsonPath, extension) + ".md"
}
