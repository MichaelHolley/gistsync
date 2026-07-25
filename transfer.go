package main

import (
	"fmt"

	"github.com/MichaelHolley/gistsync/internal/config"
	"github.com/MichaelHolley/gistsync/internal/state"
)

func trackedFile(cmd, name string) (config.File, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.File{}, fmt.Errorf("%s: %w", cmd, err)
	}
	entry, ok := cfg.Find(name)
	if !ok {
		return config.File{}, fmt.Errorf("%s: %q is not tracked — see 'gistsync list'", cmd, name)
	}
	return entry, nil
}

// refuse explains a blocked transfer in the terms the user needs mid-task: what
// diverged, where both copies live, and what each override actually does.
func refuse(cmd string, entry config.File, rec state.Record, s syncState) error {
	var cause string
	switch s {
	case stateConflict:
		cause = "both the local file and the gist changed since the last sync"
		if rec.LastSyncedHash == "" && rec.LastSyncedGistSHA == "" {
			cause = "this device has never synced it, so the local file and the gist have no common version"
		}
	case stateBehind:
		cause = "the gist changed since the last sync, so pushing would discard it"
	case stateAhead:
		cause = "the local file changed since the last sync, so pulling would discard it"
	}
	return fmt.Errorf(`refusing to %s %q — %s (%s)
  local: %s
  gist:  %s
Compare them, then choose a winner:
  gistsync push --force %s   local wins, overwrites the gist
  gistsync pull --force %s   remote wins, overwrites the local file`,
		cmd, entry.Name, s, cause, entry.Path, gistURL(rec.GistID), entry.Name, entry.Name)
}
