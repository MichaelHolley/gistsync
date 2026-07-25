// Package store resolves and creates the per-device gistsync directory.
package store

import (
	"errors"
	"os"
	"path/filepath"
)

const dirName = ".gistsync"

const emptyState = "{\n  \"files\": []\n}\n"

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, dirName), nil
}

func ConfigPath(dir string) string { return filepath.Join(dir, "config.toml") }

func StatePath(dir string) string { return filepath.Join(dir, "state.json") }

// Init creates the gistsync directory and its two files, leaving any that
// already exist untouched. It reports whether anything was created.
func Init() (dir string, created bool, err error) {
	dir, err = Dir()
	if err != nil {
		return "", false, err
	}
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return dir, false, err
	}

	configCreated, err := createIfMissing(ConfigPath(dir), "")
	if err != nil {
		return dir, false, err
	}
	stateCreated, err := createIfMissing(StatePath(dir), emptyState)
	if err != nil {
		return dir, configCreated, err
	}
	return dir, configCreated || stateCreated, nil
}

// WriteFileAtomic writes via a temp file in the destination directory and
// renames, so an interrupted write cannot truncate the existing file.
func WriteFileAtomic(path string, contents []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(contents); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func createIfMissing(path, contents string) (bool, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	if _, err := f.WriteString(contents); err != nil {
		return false, err
	}
	return true, nil
}
