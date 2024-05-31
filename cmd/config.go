package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Remotes []Remote `json:"remotes" yaml:"remotes"`
}

func (c *Config) Write() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	rcpDir := filepath.Join(configDir, "rcp")
	err = os.MkdirAll(rcpDir, 0644)
	if err != nil {
		return err
	}
	configPath := filepath.Join(rcpDir, "rcp.yaml")

	fmt.Printf("[INFO] write config to %s\n", configPath)
	file, err := os.OpenFile(configPath, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	return encoder.Encode(c)
}

func ReadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configDir, err := os.UserConfigDir()
		if os.IsNotExist(err) {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		configPath = filepath.Join(configDir, "rcp", "rcp.yaml")
	}
	file, err := os.OpenFile(configPath, os.O_RDONLY, 0644)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	cfg := &Config{}
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
