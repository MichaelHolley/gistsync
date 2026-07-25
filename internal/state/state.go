// Package state reads and writes the tool-managed state.json sync records.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/MichaelHolley/gistsync/internal/store"
)

type Record struct {
	Name              string `json:"name"`
	GistID            string `json:"gist_id"`
	LastSyncedHash    string `json:"last_synced_hash"`
	LastSyncedGistSHA string `json:"last_synced_gist_sha"`
}

type State struct {
	Files []Record `json:"files"`
}

func Load() (*State, error) {
	dir, err := store.Dir()
	if err != nil {
		return nil, err
	}
	path := store.StatePath(dir)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &State{}, nil
	}
	if err != nil {
		return nil, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &s, nil
}

func (s *State) Save() error {
	dir, err := store.Dir()
	if err != nil {
		return err
	}
	if s.Files == nil {
		s.Files = []Record{}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteFileAtomic(store.StatePath(dir), append(data, '\n'), 0o644)
}

func (s *State) Find(name string) (Record, bool) {
	for _, r := range s.Files {
		if r.Name == name {
			return r, true
		}
	}
	return Record{}, false
}

func (s *State) Put(r Record) {
	for i := range s.Files {
		if s.Files[i].Name == r.Name {
			s.Files[i] = r
			return
		}
	}
	s.Files = append(s.Files, r)
}

func (s *State) Remove(name string) bool {
	for i, r := range s.Files {
		if r.Name == name {
			s.Files = append(s.Files[:i], s.Files[i+1:]...)
			return true
		}
	}
	return false
}
