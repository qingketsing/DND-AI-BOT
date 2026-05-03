package soak

import (
	"context"
	"errors"
	"fmt"
)

const defaultRounds = 50
const FailureReasonJudgeError = "judge_error"

var ErrInvalidRunner = errors.New("invalid soak runner")

// Player generates the next simulated user input.
type Player interface {
	NextInput(ctx context.Context, input PlayerInput) (string, error)
}

// MessageSender sends one message to the real backend.
type MessageSender interface {
	SendMessage(ctx context.Context, sessionID string, content string) (MessageResult, error)
}

// Judge evaluates one backend reply.
type JudgeEvaluator interface {
	Evaluate(ctx context.Context, input JudgeInput) (JudgeResult, error)
}

// Runner coordinates player simulation, backend calls, and LLM judging.
type Runner struct {
	config        SoakConfig
	player        Player
	sender        MessageSender
	judge         JudgeEvaluator
	roundReporter RoundReporter
}

// RoundReporter observes one completed round and the current partial report.
type RoundReporter func(record RoundRecord, report SoakReport)

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithRoundReporter registers a per-round observer for logs or checkpoint writes.
func WithRoundReporter(reporter RoundReporter) RunnerOption {
	return func(runner *Runner) {
		runner.roundReporter = reporter
	}
}

// NewRunner creates a soak evaluation runner.
func NewRunner(config SoakConfig, player Player, sender MessageSender, judge JudgeEvaluator, options ...RunnerOption) *Runner {
	runner := &Runner{config: config, player: player, sender: sender, judge: judge}
	for _, option := range options {
		if option != nil {
			option(runner)
		}
	}
	return runner
}

// Run executes the configured number of long-session test rounds.
func (r *Runner) Run(ctx context.Context) (*SoakReport, error) {
	if r == nil || r.player == nil || r.sender == nil || r.judge == nil {
		return nil, ErrInvalidRunner
	}
	rounds := r.config.Rounds
	if rounds <= 0 {
		rounds = defaultRounds
	}

	records := make([]RoundRecord, 0, rounds)
	for round := 1; round <= rounds; round++ {
		userInput, err := r.player.NextInput(ctx, PlayerInput{
			Scenario:      r.config.Scenario,
			Round:         round,
			PreviousTurns: append([]RoundRecord(nil), records...),
			GameObjective: r.config.Scenario.Objective,
		})
		if err != nil {
			return nil, err
		}

		messageResult, err := r.sender.SendMessage(ctx, r.config.SessionID, userInput)
		if err != nil {
			return nil, err
		}

		judgeResult, judgeErr := r.judge.Evaluate(ctx, JudgeInput{
			Scenario:      r.config.Scenario,
			Round:         round,
			UserInput:     userInput,
			AgentReply:    messageResult.AgentReply,
			PreviousTurns: append([]RoundRecord(nil), records...),
			HTTPStatus:    messageResult.HTTPStatus,
			LatencyMS:     messageResult.LatencyMS,
		})
		if judgeErr != nil {
			judgeResult = JudgeResult{
				Success:        false,
				Score:          0,
				FailureReasons: []string{FailureReasonJudgeError},
				Comment:        fmt.Sprintf("judge evaluation failed: %v", judgeErr),
			}
		}

		records = append(records, RoundRecord{
			Round:          round,
			UserInput:      userInput,
			AgentReply:     messageResult.AgentReply,
			LatencyMS:      messageResult.LatencyMS,
			HTTPStatus:     messageResult.HTTPStatus,
			Success:        judgeResult.Success,
			Score:          judgeResult.Score,
			FailureReasons: append([]string(nil), judgeResult.FailureReasons...),
			JudgeComment:   judgeResult.Comment,
		})
		if r.roundReporter != nil {
			r.roundReporter(records[len(records)-1], BuildReport(r.config.SessionID, records))
		}
	}

	report := BuildReport(r.config.SessionID, records)
	return &report, nil
}
