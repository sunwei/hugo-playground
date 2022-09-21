package config

import (
	"github.com/spf13/afero"
	"github.com/sunwei/hugo-playground/parser/metadecoders"
)

var (
	ValidConfigFileExtensions = []string{"toml"}
)

// FromFileToMap is the same as FromFile, but it returns the config values
// as a simple map.
func FromFileToMap(fs afero.Fs, filename string) (map[string]any, error) {
	return loadConfigFromFile(fs, filename)
}

func loadConfigFromFile(fs afero.Fs, filename string) (map[string]any, error) {
	m, err := metadecoders.Default.UnmarshalFileToMap(fs, filename)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// FromFile loads the configuration from the given filename.
func FromFile(fs afero.Fs, filename string) (Provider, error) {
	m, err := loadConfigFromFile(fs, filename)
	if err != nil {
		return nil, err
	}
	return NewFrom(m), nil
}
