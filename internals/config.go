package internals

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"gopkg.in/yaml.v2"
)

type SplitbitConfig struct {
	Name      string          `yaml:"name"`
	Env       string          `yaml:"env"`
	Algorithm string          `yaml:"algorithm"`
	Scheme    string          `yaml:"scheme"`
	Backends  []BackendConfig `yaml:"backends"`
}

type BackendConfig struct {
	Name        string `yaml:"name"`
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Weight      int    `yaml:"weight"`
	HealthCheck string `yaml:"health_check"`
}

func (cfg *SplitbitConfig) Validate() error {
	if cfg.Name == "" {
		return errors.New("name is required for the configuration")
	}

	envs := []string{EnvDev, EnvProd}
	if !slices.Contains(envs, cfg.Env) {
		if cfg.Env == "" {
			cfg.Env = EnvDev
		} else {
			return errors.New("only [DEV, PROD] are supported as env")
		}
	}

	supported := []string{"round-robin", "weighted-round-robin"}
	if !slices.Contains(supported, cfg.Algorithm) {
		return errors.New("only [round-robin, weighted-round-robin] are supported as algorithm")
	}

	schemes := []string{"tcp"}
	if !slices.Contains(schemes, cfg.Scheme) {
		return errors.New("only [tcp] scheme are supported as backends")
	}

	if len(cfg.Backends) == 0 {
		return errors.New("at least one backend is required")
	}

	for i, backend := range cfg.Backends {
		if err := backend.Validate(); err != nil {
			return fmt.Errorf("backend %d (%s): %w", i, backend.Name, err)
		}
	}

	return nil
}

func (cfg *BackendConfig) Validate() error {
	if cfg.Name == "" {
		return errors.New("name is required for the configuration")
	}

	if cfg.Host == "" {
		return errors.New("host is required for the configuration")
	}

	if cfg.Port > 65535 || cfg.Port < 1 {
		return errors.New("a valid port is required for the configuration")
	}

	if cfg.HealthCheck == "" {
		return errors.New("health_check is required for the configuration")
	}

	if cfg.Weight < 0 {
		errMsg := fmt.Sprintf("weight must be a positive integer, found %d for %s\n", cfg.Weight, cfg.Name)
		return errors.New(errMsg)
	}

	if cfg.Weight == 0 {
		cfg.Weight = 1
	}

	return nil
}

// LoadConfig loads the configuration into a struct and returns it
func LoadConfig(path string) (*SplitbitConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg SplitbitConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
