// Package terrier reads the repo registry kept by terrier
// (https://github.com/dittofleet/terrier), so a repo registered there is a
// project here without being registered a second time.
//
// Terrier's CLI is its contract: the registry file is private and its
// layout can change, so everything comes from `terrier ls --json`.
//
// Nothing here is required. A machine without terrier, or with one too
// new to be understood, is a machine where whatagain's projects are the
// ones in its own store and nothing else.
package terrier

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
)

const binary = "terrier"

// project is the record terrier prints. Only the fields whatagain acts on
// are named: terrier only ever adds fields, so the rest are ignored here
// rather than tracked.
type project struct {
	// Slug is the "owner/name" of the repo's GitHub origin, absent when it
	// has neither, which is a repo whatagain has no way to name.
	Slug string `json:"slug"`
	// Missing marks a project whose directory is gone. Terrier reports
	// those for `terrier prune` to explain. Here they are skipped.
	Missing bool `json:"missing"`
}

// The registry is read once per process. Several commands ask for it, and
// none of them wants a second subprocess for an answer that cannot have
// changed in between.
var cached = sync.OnceValue(load)

// Slugs returns the "owner/name" of every repo terrier has registered.
func Slugs() []string {
	return cached()
}

// Has reports whether terrier has a repo registered under id. GitHub
// treats owner and repo names case-insensitively, so this does too, and
// agrees with how the store compares project ids.
func Has(id string) bool {
	return containsFold(Slugs(), id)
}

func load() []string {
	if _, err := exec.LookPath(binary); err != nil {
		// Terrier is optional, and its absence is not something to report:
		// whatagain's own store is the whole list, exactly as before.
		return nil
	}
	// The version gate and the registry do not depend on each other, and
	// each is a process launch, which is nearly all of what asking terrier
	// anything costs. Both are started at once and the gate still decides
	// afterwards whether the answer may be used: a terrier too new to
	// understand is the rare case, and all it wastes there is a listing
	// nobody reads.
	type listing struct {
		out []byte
		err error
	}
	listed := make(chan listing, 1)
	go func() {
		out, err := run("ls", "--json")
		listed <- listing{out, err}
	}()

	versionErr := checkVersion()
	got := <-listed

	switch {
	case versionErr != nil:
		warn(versionErr)
		return nil
	case got.err != nil:
		warn(got.err)
		return nil
	}
	var registry struct {
		Projects []project `json:"projects"`
	}
	if err := json.Unmarshal(got.out, &registry); err != nil {
		warn(fmt.Errorf("could not read the terrier registry: %w", err))
		return nil
	}
	return collect(registry.Projects)
}

// collect keeps the registered repos whatagain can name, and drops the
// duplicates: the same repo cloned to two paths is two projects to
// terrier, which records paths, and one project here, which records repos.
func collect(projects []project) []string {
	var found []string
	for _, p := range projects {
		if p.Missing || p.Slug == "" {
			continue
		}
		if !containsFold(found, p.Slug) {
			found = append(found, p.Slug)
		}
	}
	return found
}

func containsFold(ids []string, id string) bool {
	return slices.ContainsFunc(ids, func(s string) bool { return strings.EqualFold(s, id) })
}

// run executes a terrier command and returns its standard output. Its
// standard error is captured rather than inherited, so a terrier that
// fails cannot print over whatagain's output before whatagain has decided
// what to say about it.
func run(args ...string) ([]byte, error) {
	out, err := exec.Command(binary, args...).Output()
	if err == nil {
		return out, nil
	}
	command := binary + " " + strings.Join(args, " ")
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if stderr := bytes.TrimSpace(exit.Stderr); len(stderr) > 0 {
			return nil, fmt.Errorf("`%s` failed: %s", command, stderr)
		}
	}
	return nil, fmt.Errorf("`%s` failed: %w", command, err)
}

// warn reports a terrier that is installed but cannot be used. Staying
// quiet would be worse than the noise: the projects it holds would simply
// be missing from the listing, with nothing on screen to say why.
func warn(err error) {
	fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
}
