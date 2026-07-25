// Package gh wraps the GitHub CLI, the tool's only route to GitHub.
package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const descriptionPrefix = "gistsync:"

func Description(name string) string { return descriptionPrefix + name }

type Gist struct {
	ID      string `json:"id"`
	HTMLURL string `json:"html_url"`
	Files   map[string]struct {
		Content   string `json:"content"`
		Truncated bool   `json:"truncated"`
	} `json:"files"`
	History []struct {
		Version string `json:"version"`
	} `json:"history"`
}

// Content returns the gist file stored under name, byte-exact.
func (g Gist) Content(name string) ([]byte, error) {
	f, ok := g.Files[name]
	if !ok {
		return nil, fmt.Errorf("gist %s has no file named %q", g.ID, name)
	}
	if f.Truncated {
		return nil, fmt.Errorf("gist %s is too large for gistsync to transfer", g.ID)
	}
	return []byte(f.Content), nil
}

// Version is the gist's current commit SHA.
func (g Gist) Version() string {
	if len(g.History) == 0 {
		return ""
	}
	return g.History[0].Version
}

// Preflight fails with an actionable message when gh cannot reach GitHub on
// the user's behalf. Every networked command calls it first.
func Preflight() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("the GitHub CLI ('gh') is not installed — see https://cli.github.com")
	}
	if _, err := run(nil, "auth", "status"); err != nil {
		return fmt.Errorf("gh cannot reach GitHub as an authenticated user — run 'gh auth login' if you are not logged in\n%w", err)
	}
	return nil
}

// FindGistID returns the ID of the gist whose description is
// gistsync:<name>, or an empty string when the user has no such gist.
func FindGistID(name string) (string, error) {
	out, err := run(nil, "api", "--paginate", "/gists?per_page=100",
		"--jq", fmt.Sprintf(`.[] | select(.description == %q) | .id`, Description(name)))
	if err != nil {
		return "", err
	}
	ids := strings.Fields(string(out))
	if len(ids) == 0 {
		return "", nil
	}
	return ids[0], nil
}

func GetGist(id string) (Gist, error) {
	return apiGist(nil, "api", "/gists/"+id)
}

func CreateGist(name string, content []byte) (Gist, error) {
	body, err := gistBody(name, content, Description(name))
	if err != nil {
		return Gist{}, err
	}
	return apiGist(body, "api", "-X", "POST", "/gists", "--input", "-")
}

func UpdateGist(id, name string, content []byte) (Gist, error) {
	body, err := gistBody(name, content, "")
	if err != nil {
		return Gist{}, err
	}
	return apiGist(body, "api", "-X", "PATCH", "/gists/"+id, "--input", "-")
}

func gistBody(name string, content []byte, description string) ([]byte, error) {
	payload := map[string]any{
		"files": map[string]any{
			name: map[string]string{"content": string(content)},
		},
	}
	if description != "" {
		payload["description"] = description
		payload["public"] = false
	}
	return json.Marshal(payload)
}

func apiGist(body []byte, args ...string) (Gist, error) {
	out, err := run(body, args...)
	if err != nil {
		return Gist{}, err
	}
	var g Gist
	if err := json.Unmarshal(out, &g); err != nil {
		return Gist{}, fmt.Errorf("unexpected response from gh: %w", err)
	}
	return g, nil
}

func run(stdin []byte, args ...string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("gh %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}
