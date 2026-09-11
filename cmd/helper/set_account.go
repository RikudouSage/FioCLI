package helper

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

func SetCurrentAccount(filepath string, account string) error {
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed opening %s: %w", filepath, err)
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed reading %s: %w", filepath, err)
	}

	type config struct {
		Database       string `yaml:"database,omitempty"`
		CurrentAccount string `yaml:"current-account,omitempty"`
	}
	var cfg config
	if err := yaml.Unmarshal(fileContent, &cfg); err != nil {
		return fmt.Errorf("failed parsing %s: %w", filepath, err)
	}

	cfg.CurrentAccount = account
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed marshalling %s: %w", filepath, err)
	}

	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("failed seeking %s: %w", filepath, err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed seeking %s: %w", filepath, err)
	}
	if _, err := file.Write(out); err != nil {
		return fmt.Errorf("failed writing %s: %w", filepath, err)
	}

	return nil
}
