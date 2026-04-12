package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ForwardEntry struct {
	Protocol   string   `yaml:"protocol"`
	ListenPort int      `yaml:"listen_port"`
	RemoteHost string   `yaml:"remote_host"`
	RemotePort int      `yaml:"remote_port"`
	TLSCert    string   `yaml:"tls_cert,omitempty"`
	TLSKey     string   `yaml:"tls_key,omitempty"`
	AllowCIDRs []string `yaml:"allow_cidrs,omitempty"`
	DenyCIDRs  []string `yaml:"deny_cidrs,omitempty"`
	RateLimit  int      `yaml:"rate_limit,omitempty"`
}

type Config struct {
	Forwards map[string]ForwardEntry `yaml:"forwards"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".portfwd.yml")
}

func Load() (*Config, error) {
	cfg := &Config{
		Forwards: make(map[string]ForwardEntry),
	}

	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Forwards == nil {
		cfg.Forwards = make(map[string]ForwardEntry)
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}
