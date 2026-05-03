package soak

import (
	"context"
	"errors"
)

const defaultRounds = 50

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
	config SoakConfig
	player Player
	sender MessageSender
	judge  JudgeEvaluator
}

// NewRunner creates a soak evaluation runner.
func NewRunner(config SoakConfig, player Player, sender MessageSender, judge JudgeEvaluator) *Runner {
	return &Runner{config: config, player: player, sender: sender, judge: judge}
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

		judgeResult, err := r.judge.Evaluate(ctx, JudgeInput{
			Scenario:      r.config.Scenario,
			Round:         round,
			UserInput:     userInput,
			AgentReply:    messageResult.AgentReply,
			PreviousTurns: append([]RoundRecord(nil), records...),
			HTTPStatus:    messageResult.HTTPStatus,
			LatencyMS:     messageResult.LatencyMS,
		})
		if err != nil {
			return nil, err
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
	}

	report := BuildReport(r.config.SessionID, records)
	return &report, nil
}
