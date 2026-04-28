package runtime

import "time"

type RuntimeModelCallLogMode string

const (
	RuntimeModelCallLogOff  RuntimeModelCallLogMode = "off"
	RuntimeModelCallLogSlow RuntimeModelCallLogMode = "slow"
	RuntimeModelCallLogAll  RuntimeModelCallLogMode = "all"
)

type RuntimeModelCallLogConfig struct {
	Mode      RuntimeModelCallLogMode
	Threshold time.Duration
}

func DefaultRuntimeModelCallLogConfig() RuntimeModelCallLogConfig {
	return RuntimeModelCallLogConfig{
		Mode:      RuntimeModelCallLogSlow,
		Threshold: time.Second,
	}
}

func normalizeRuntimeModelCallLogConfig(config RuntimeModelCallLogConfig) RuntimeModelCallLogConfig {
	if config.Mode != RuntimeModelCallLogOff && config.Mode != RuntimeModelCallLogSlow && config.Mode != RuntimeModelCallLogAll {
		config.Mode = RuntimeModelCallLogSlow
	}
	if config.Threshold < 0 {
		config.Threshold = time.Second
	}
	return config
}
