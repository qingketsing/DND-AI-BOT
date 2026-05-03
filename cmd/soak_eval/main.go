package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"DND-AI-BOT/internal/eval/soak"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		log.Fatalf("soak eval failed: %v", err)
	}
}

func run(ctx context.Context, args []string) error {
	configPath, outputPath, err := parseArgs(args)
	if err != nil {
		return err
	}
	config, err := soak.LoadConfig(configPath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(outputPath) != "" {
		config.OutputPath = outputPath
	}

	report, err := soak.RunSoakEval(ctx, config)
	if err != nil {
		return err
	}
	fmt.Printf("soak eval completed: session_id=%s rounds=%d success_rate=%.2f output=%s\n",
		report.SessionID,
		report.Rounds,
		report.SuccessRate,
		config.OutputPath,
	)
	return nil
}

func parseArgs(args []string) (string, string, error) {
	fs := flag.NewFlagSet("soak_eval", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var configPath string
	var outputPath string
	fs.StringVar(&configPath, "config", filepath.Join("configs", "eval", "soak_the_city_50.json"), "path to soak eval JSON config")
	fs.StringVar(&outputPath, "output", "", "optional JSON report output path")
	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(configPath) == "" {
		return "", "", fmt.Errorf("config path is required")
	}
	return configPath, outputPath, nil
}
