package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	Daemon   DaemonConfig   `toml:"daemon"`
	Discord  DiscordConfig  `toml:"discord"`
	Defaults DefaultsConfig `toml:"defaults"`
}

type DaemonConfig struct {
	CheckInterval string `toml:"check_interval"`
	IdleThreshold string `toml:"idle_threshold"`
	AutoRestart   bool   `toml:"auto_restart"`
	MaxRestarts   int    `toml:"max_restarts"`
	MaxLogAgeDays int    `toml:"max_log_age_days"`
	HookPort      int    `toml:"hook_port"`
}

type DiscordConfig struct {
	WebhookURL string `toml:"webhook_url"`
}

type DefaultsConfig struct {
	Agent string `toml:"agent"`
}

func (c *DaemonConfig) CheckIntervalDuration() time.Duration {
	d, err := time.ParseDuration(c.CheckInterval)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

func (c *DaemonConfig) GetHookPort() int {
	if c.HookPort <= 0 {
		return 9111
	}
	return c.HookPort
}

func (c *DaemonConfig) MaxLogAge() int {
	if c.MaxLogAgeDays <= 0 {
		return 7
	}
	return c.MaxLogAgeDays
}

func (c *DaemonConfig) IdleThresholdDuration() time.Duration {
	d, err := time.ParseDuration(c.IdleThreshold)
	if err != nil {
		return 5 * time.Minute
	}
	return d
}

func Default() *Config {
	return &Config{
		Daemon: DaemonConfig{
			CheckInterval: "30s",
			IdleThreshold: "5m",
			AutoRestart:   true,
			MaxRestarts:   3,
		},
		Discord: DiscordConfig{},
		Defaults: DefaultsConfig{
			Agent: "claude",
		},
	}
}

func DataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ina")
}

func PidFile() string {
	return filepath.Join(DataDir(), "ina.pid")
}

func SocketPath() string {
	return filepath.Join(DataDir(), "ina.sock")
}

func LogFile() string {
	return filepath.Join(DataDir(), "ina.log")
}

func ConfigPath() string {
	return filepath.Join(DataDir(), "config.toml")
}

func RegistryFile() string {
	return filepath.Join(DataDir(), "registry.json")
}

func EnsureDir() error {
	return os.MkdirAll(DataDir(), 0700)
}

func Load() (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

