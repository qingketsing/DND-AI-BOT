package soak

import (
	"context"
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
	inputs  []JudgeInput
}

func (f *fakeJudge) Evaluate(ctx context.Context, input JudgeInput) (JudgeResult, error) {
	_ = ctx
	f.inputs = append(f.inputs, input)
	output := f.outputs[0]
	f.outputs = f.outputs[1:]
	return output, nil
}
