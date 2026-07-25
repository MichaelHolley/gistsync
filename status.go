package main

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/MichaelHolley/gistsync/internal/config"
	"github.com/MichaelHolley/gistsync/internal/gh"
	"github.com/MichaelHolley/gistsync/internal/state"
)

type syncState string

const (
	stateClean       syncState = "clean"
	stateAhead       syncState = "ahead"
	stateBehind      syncState = "behind"
	stateConflict    syncState = "conflict"
	stateNeverPushed syncState = "never pushed"
	stateMissing     syncState = "missing locally"
)

// classify implements the state table: what changed since the last successful
// sync, on each side, decides the state.
func classify(rec state.Record, localHash, gistSHA string) syncState {
	localChanged := localHash != rec.LastSyncedHash
	remoteChanged := gistSHA != rec.LastSyncedGistSHA
	switch {
	case localChanged && remoteChanged:
		return stateConflict
	case localChanged:
		return stateAhead
	case remoteChanged:
		return stateBehind
	default:
		return stateClean
	}
}

func runStatus(args []string) error {
	fs := newFlagSet("status")
	if err := parseFlags(fs, args, "Usage: gistsync status"); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("status: unexpected argument %q", fs.Arg(0))
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}
	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}

	if len(cfg.Files) == 0 {
		fmt.Println("No files tracked — add one with 'gistsync add <path>'.")
		return nil
	}

	online := gh.Preflight() == nil
	if !online {
		fmt.Fprintln(os.Stderr, "gistsync: GitHub is unreachable — showing local information only")
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATE\tPATH")
	for _, f := range cfg.Files {
		rec, _ := st.Find(f.Name)
		fmt.Fprintf(w, "%s\t%s\t%s\n", f.Name, describe(f, rec, online), f.Path)
	}
	return w.Flush()
}

func describe(f config.File, rec state.Record, online bool) string {
	localHash, err := hashFile(f.Path)
	if errors.Is(err, os.ErrNotExist) {
		return string(stateMissing)
	}
	if err != nil {
		return fmt.Sprintf("unreadable (%v)", err)
	}
	if rec.GistID == "" {
		return string(stateNeverPushed)
	}
	if !online {
		return "remote unknown, " + localSummary(localHash, rec)
	}

	gist, err := gh.GetGist(rec.GistID)
	if err != nil {
		return "remote unknown, " + localSummary(localHash, rec)
	}
	return string(classify(rec, localHash, gist.Version()))
}

func localSummary(localHash string, rec state.Record) string {
	if localHash != rec.LastSyncedHash {
		return "local changed"
	}
	return "local unchanged"
}

func hashFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return hashContent(content), nil
}
