package ratelimit

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Enabled bool

	LoginIPLimit       int
	LoginIPWindow      time.Duration
	LoginAccountLimit  int
	LoginAccountWindow time.Duration

	RegisterIPLimit     int
	RegisterIPWindow    time.Duration
	RegisterEmailLimit  int
	RegisterEmailWindow time.Duration

	MessageUserLimit     int
	MessageUserWindow    time.Duration
	MessageSessionLimit  int
	MessageSessionWindow time.Duration
	MessageIPLimit       int
	MessageIPWindow      time.Duration
}

func DefaultConfig() Config {
	return Config{
		Enabled: true,

		LoginIPLimit:       10,
		LoginIPWindow:      time.Minute,
		LoginAccountLimit:  5,
		LoginAccountWindow: 5 * time.Minute,

		RegisterIPLimit:     5,
		RegisterIPWindow:    time.Hour,
		RegisterEmailLimit:  3,
		RegisterEmailWindow: time.Hour,

		MessageUserLimit:     30,
		MessageUserWindow:    time.Minute,
		MessageSessionLimit:  20,
		MessageSessionWindow: time.Minute,
		MessageIPLimit:       60,
		MessageIPWindow:      time.Minute,
	}
}

func LoadConfigFromEnv() Config {
	config := DefaultConfig()
	config.Enabled = envBool("RATE_LIMIT_ENABLED", config.Enabled)
	config.LoginIPLimit = envInt("RATE_LIMIT_LOGIN_IP_LIMIT", config.LoginIPLimit)
	config.LoginIPWindow = envDuration("RATE_LIMIT_LOGIN_IP_WINDOW", config.LoginIPWindow)
	config.LoginAccountLimit = envInt("RATE_LIMIT_LOGIN_ACCOUNT_LIMIT", config.LoginAccountLimit)
	config.LoginAccountWindow = envDuration("RATE_LIMIT_LOGIN_ACCOUNT_WINDOW", config.LoginAccountWindow)
	config.RegisterIPLimit = envInt("RATE_LIMIT_REGISTER_IP_LIMIT", config.RegisterIPLimit)
	config.RegisterIPWindow = envDuration("RATE_LIMIT_REGISTER_IP_WINDOW", config.RegisterIPWindow)
	config.RegisterEmailLimit = envInt("RATE_LIMIT_REGISTER_EMAIL_LIMIT", config.RegisterEmailLimit)
	config.RegisterEmailWindow = envDuration("RATE_LIMIT_REGISTER_EMAIL_WINDOW", config.RegisterEmailWindow)
	config.MessageUserLimit = envInt("RATE_LIMIT_MESSAGE_USER_LIMIT", config.MessageUserLimit)
	config.MessageUserWindow = envDuration("RATE_LIMIT_MESSAGE_USER_WINDOW", config.MessageUserWindow)
	config.MessageSessionLimit = envInt("RATE_LIMIT_MESSAGE_SESSION_LIMIT", config.MessageSessionLimit)
	config.MessageSessionWindow = envDuration("RATE_LIMIT_MESSAGE_SESSION_WINDOW", config.MessageSessionWindow)
	config.MessageIPLimit = envInt("RATE_LIMIT_MESSAGE_IP_LIMIT", config.MessageIPLimit)
	config.MessageIPWindow = envDuration("RATE_LIMIT_MESSAGE_IP_WINDOW", config.MessageIPWindow)
	return config
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
