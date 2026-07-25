package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

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

	type row struct{ name, state, path string }
	rows := make([]row, 0, len(cfg.Files))
	nameW, stateW := utf8.RuneCountInString("NAME"), utf8.RuneCountInString("STATE")
	for _, f := range cfg.Files {
		rec, _ := st.Find(f.Name)
		r := row{f.Name, describe(f, rec, online), f.Path}
		nameW = max(nameW, utf8.RuneCountInString(r.name))
		stateW = max(stateW, utf8.RuneCountInString(r.state))
		rows = append(rows, r)
	}

	fmt.Printf("%-*s  %-*s  %s\n", nameW, "NAME", stateW, "STATE", "PATH")
	for _, r := range rows {
		fmt.Printf("%-*s  %s  %s\n", nameW, r.name, colorState(r.state, stateW), r.path)
	}
	return nil
}

var stateColor = map[syncState]string{
	stateClean:       "\x1b[32m",
	stateAhead:       "\x1b[33m",
	stateBehind:      "\x1b[36m",
	stateConflict:    "\x1b[31m",
	stateNeverPushed: "\x1b[35m",
	stateMissing:     "\x1b[31m",
}

// useColor is resolved once: colour only when stdout is a terminal, so pipes
// and redirects stay plain text.
var useColor = os.Getenv("NO_COLOR") == "" && isTerminal(os.Stdout)

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// colorState pads before wrapping so the escape bytes never count towards
// column width.
func colorState(s string, width int) string {
	padded := s + strings.Repeat(" ", max(0, width-utf8.RuneCountInString(s)))
	c, ok := stateColor[syncState(s)]
	if !ok || !useColor {
		return padded
	}
	return c + padded + "\x1b[0m"
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
