package soak

import (
	"context"
	"errors"
	"testing"
)

func TestRunnerExecutesConfiguredRoundsAndBuildsReport(t *testing.T) {
	player := &fakePlayer{
		outputs: []string{"创建角色", "继续探索"},
	}
	sender := &fakeMessageSender{
		outputs: []MessageResult{
			{AgentReply: "角色已创建", HTTPStatus: 200, LatencyMS: 100},
			{AgentReply: "你继续探索", HTTPStatus: 200, LatencyMS: 200},
		},
	}
	judge := &fakeJudge{
		outputs: []JudgeResult{
			{Success: true, Score: 1},
			{Success: false, Score: 0.4, FailureReasons: []string{"forgot_scene"}, Comment: "场景漂移"},
		},
	}

	report, err := NewRunner(SoakConfig{
		SessionID: "session-1",
		Rounds:    2,
		Scenario:  ScenarioConfig{Objective: "测试长会话"},
	}, player, sender, judge).Run(context.Background())
	if err != nil {
		t.Fatalf("expected runner to succeed, got %v", err)
	}

	if report.Rounds != 2 {
		t.Fatalf("expected 2 rounds, got %d", report.Rounds)
	}
	if report.SuccessRounds != 1 {
		t.Fatalf("expected 1 successful round, got %d", report.SuccessRounds)
	}
	if len(report.Records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(report.Records))
	}
	if report.Records[1].FailureReasons[0] != "forgot_scene" {
		t.Fatalf("expected failure reason forgot_scene, got %v", report.Records[1].FailureReasons)
	}
	if len(player.inputs[1].PreviousTurns) != 1 {
		t.Fatalf("expected second player call to receive previous turn, got %d", len(player.inputs[1].PreviousTurns))
	}
}

func TestRunnerRejectsInvalidDependencies(t *testing.T) {
	_, err := NewRunner(SoakConfig{SessionID: "session-1", Rounds: 1}, nil, nil, nil).Run(context.Background())

	if err != ErrInvalidRunner {
		t.Fatalf("expected ErrInvalidRunner, got %v", err)
	}
}

func TestRunnerRecordsJudgeErrorAsFailedRoundAndContinues(t *testing.T) {
	player := &fakePlayer{outputs: []string{"继续探索", "检查门"}}
	sender := &fakeMessageSender{outputs: []MessageResult{
		{AgentReply: "你继续探索", HTTPStatus: 200, LatencyMS: 100},
		{AgentReply: "门后有声音", HTTPStatus: 200, LatencyMS: 200},
	}}
	judge := &fakeJudge{
		errs: []error{errors.New("parse judge JSON: unexpected end of JSON input"), nil},
		outputs: []JudgeResult{
			{Success: true, Score: 0.9},
		},
	}

	report, err := NewRunner(SoakConfig{SessionID: "session-1", Rounds: 2}, player, sender, judge).Run(context.Background())
	if err != nil {
		t.Fatalf("expected runner to continue after judge error, got %v", err)
	}

	if report.Rounds != 2 {
		t.Fatalf("expected 2 rounds, got %d", report.Rounds)
	}
	first := report.Records[0]
	if first.Success {
		t.Fatal("expected judge error round to be marked failed")
	}
	if first.Score != 0 {
		t.Fatalf("expected score 0 for judge error, got %f", first.Score)
	}
	if first.FailureReasons[0] != FailureReasonJudgeError {
		t.Fatalf("expected judge error failure reason, got %v", first.FailureReasons)
	}
	if report.SuccessRounds != 1 {
		t.Fatalf("expected one success after continuing, got %d", report.SuccessRounds)
	}
}

func TestRunnerCallsRoundReporterAfterEachRound(t *testing.T) {
	player := &fakePlayer{outputs: []string{"继续探索"}}
	sender := &fakeMessageSender{outputs: []MessageResult{{AgentReply: "你继续探索", HTTPStatus: 200, LatencyMS: 100}}}
	judge := &fakeJudge{outputs: []JudgeResult{{Success: true, Score: 1}}}
	var reports []RoundRecord

	report, err := NewRunner(SoakConfig{SessionID: "session-1", Rounds: 1}, player, sender, judge, WithRoundReporter(func(record RoundRecord, report SoakReport) {
		reports = append(reports, record)
		if report.Rounds != 1 {
			t.Fatalf("expected checkpoint report rounds 1, got %d", report.Rounds)
		}
	})).Run(context.Background())
	if err != nil {
		t.Fatalf("expected runner to succeed, got %v", err)
	}

	if len(reports) != 1 {
		t.Fatalf("expected one round report, got %d", len(reports))
	}
	if report.SuccessRate != 1 {
		t.Fatalf("expected success rate 1, got %f", report.SuccessRate)
	}
}

type fakePlayer struct {
	outputs []string
	inputs  []PlayerInput
}

func (f *fakePlayer) NextInput(ctx context.Context, input PlayerInput) (string, error) {
	_ = ctx
	f.inputs = append(f.inputs, input)
	output := f.outputs[0]
	f.outputs = f.outputs[1:]
	return output, nil
}

type fakeMessageSender struct {
	outputs []MessageResult
	inputs  []string
}

func (f *fakeMessageSender) SendMessage(ctx context.Context, sessionID string, content string) (MessageResult, error) {
	_ = ctx
	_ = sessionID
	f.inputs = append(f.inputs, content)
	output := f.outputs[0]
	f.outputs = f.outputs[1:]
	return output, nil
}

type fakeJudge struct {
	outputs []JudgeResult
	errs    []error
	inputs  []JudgeInput
}

func (f *fakeJudge) Evaluate(ctx context.Context, input JudgeInput) (JudgeResult, error) {
	_ = ctx
	f.inputs = append(f.inputs, input)
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		if err != nil {
			return JudgeResult{}, err
		}
	}
	output := f.outputs[0]
	f.outputs = f.outputs[1:]
	return output, nil
}
