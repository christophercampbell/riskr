package config

import (
	"encoding/json"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel        string             `yaml:"log_level" json:"log_level"`
	NATS            NATS               `yaml:"nats" json:"nats"`
	HTTP            HTTP               `yaml:"http" json:"http"`
	Policy          Policy             `yaml:"policy" json:"policy"`
	Sanctions       Sanctions          `yaml:"sanctions" json:"sanctions"`
	Assets          map[string]float64 `yaml:"assets" json:"assets"`
	LatencyBudgetMS int                `yaml:"latency_budget_ms" json:"latency_budget_ms"`
}

type NATS struct {
	URLs          []string `yaml:"urls" json:"urls"`
	EnsureStreams bool     `yaml:"ensure_streams" json:"ensure_streams"`
}

type HTTP struct {
	ListenAddr     string `yaml:"listen_addr" json:"listen_addr"`
	ReadTimeoutMS  int    `yaml:"read_timeout_ms" json:"read_timeout_ms"`
	WriteTimeoutMS int    `yaml:"write_timeout_ms" json:"write_timeout_ms"`
}

type Policy struct {
	File string `yaml:"file" json:"file"`
}

type Sanctions struct {
	File string `yaml:"file" json:"file"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		// env var fallback
		path = os.Getenv("RISKR_CONFIG")
	}
	if path == "" {
		path = defaultPath()
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	applyEnvOverrides(&c)
	return &c, nil
}

func defaultPath() string {
	h, _ := os.UserHomeDir()
	return h + "/.config/riskr/config.yaml"
}

func applyEnvOverrides(c *Config) {
	if v := os.Getenv("RISKR_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("RISKR_NATS_URLS"); v != "" {
		c.NATS.URLs = strings.Split(v, ",")
	}
	// add more as needed
}

func (c *Config) JSON() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return string(b)
}
