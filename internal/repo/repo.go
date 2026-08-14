// Package repo derives the current project id from the surrounding git
// repository. Every lookup goes through `git` itself rather than through
// .git files, so linked worktrees resolve exactly like a primary checkout.
package repo

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNoProject means the current directory does not identify a GitHub
// repo, for any of the reasons below. Callers turn it into advice about
// passing --project explicitly.
var ErrNoProject = errors.New("no current project")

// Current returns the "owner/name" id of the repository containing the
// working directory, taken from its `origin` remote.
//
// The happy path is one `git` invocation: asking for the remote fails
// outside a repository too. Telling those cases apart is only worth a
// second subprocess once something has already gone wrong.
func Current() (string, error) {
	url, err := run("remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrNoProject, diagnose(err))
	}
	id, err := ParseGitHubURL(url)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrNoProject, err)
	}
	return id, nil
}

// diagnose explains why asking git for the remote failed.
func diagnose(err error) string {
	if errors.Is(err, exec.ErrNotFound) {
		return "git is not installed"
	}
	if out, err := run("rev-parse", "--is-inside-work-tree"); err != nil || out != "true" {
		return "not inside a git repository"
	}
	return "this repository has no `origin` remote"
}

func run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ParseGitHubURL extracts "owner/name" from any of the remote URL forms
// git prints: scp-style (git@github.com:owner/name.git), https, ssh://,
// and git://.
func ParseGitHubURL(url string) (string, error) {
	fail := fmt.Errorf("the `origin` remote is not a GitHub URL: %s", url)

	rest := url
	if _, after, found := strings.Cut(rest, "://"); found {
		rest = after
	}
	// Whichever form the URL took, the host ends at the first ':' or '/'.
	// Isolating it rather than searching the whole URL for "github.com"
	// keeps a host like notgithub.com from passing as the real one.
	sep := strings.IndexAny(rest, ":/")
	if sep < 0 {
		return "", fail
	}
	host, path := rest[:sep], rest[sep+1:]
	if _, after, found := strings.Cut(host, "@"); found {
		host = after
	}
	if !strings.EqualFold(host, "github.com") {
		return "", fail
	}

	path = strings.TrimSuffix(strings.TrimRight(path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fail
	}
	return parts[0] + "/" + parts[1], nil
}
