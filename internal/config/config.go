// Package config reads and writes the per-device config.toml tracked set.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/MichaelHolley/gistsync/internal/store"
)

type File struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
}

type Config struct {
	Files []File `toml:"file"`
}

var ErrNotInitialised = errors.New("no config found — run 'gistsync init' first")

func Load() (*Config, error) {
	dir, err := store.Dir()
	if err != nil {
		return nil, err
	}
	path := store.ConfigPath(dir)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotInitialised
	}
	if err != nil {
		return nil, err
	}

	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &c, nil
}

func (c *Config) Save() error {
	dir, err := store.Dir()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if len(c.Files) > 0 {
		if err := toml.NewEncoder(&buf).Encode(c); err != nil {
			return err
		}
	}
	return store.WriteFileAtomic(store.ConfigPath(dir), buf.Bytes(), 0o644)
}

func (c *Config) Find(name string) (File, bool) {
	for _, f := range c.Files {
		if f.Name == name {
			return f, true
		}
	}
	return File{}, false
}

func (c *Config) Add(f File) {
	c.Files = append(c.Files, f)
}

func (c *Config) Remove(name string) bool {
	for i, f := range c.Files {
		if f.Name == name {
			c.Files = append(c.Files[:i], c.Files[i+1:]...)
			return true
		}
	}
	return false
}
