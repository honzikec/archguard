package config

import (
	"fmt"
	"os"
)

func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s not found", path)
	}
	return &Config{}, nil
}
