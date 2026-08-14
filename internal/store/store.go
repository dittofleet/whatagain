// Package store reads and writes the single JSON file holding every
// project and item. The file lives in the config directory so it rides
// along with whatever syncs dotfiles between machines.
package store

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/dittofleet/whatagain/internal/xdg"
)

const SchemaVersion = 1

type Item struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Created string `json:"created"`
}

type Project struct {
	// ID is a GitHub repo slug, e.g. "dittofleet/shigoto-no-mori".
	ID    string `json:"id"`
	Items []Item `json:"items"`
}

type Store struct {
	SchemaVersion int        `json:"schemaVersion"`
	Projects      []*Project `json:"projects"`
}

// Path returns the location of the store file.
func Path() string {
	return filepath.Join(xdg.ConfigDir(xdg.App), "todo.json")
}

// Load reads the store file. A missing file is not an error: it yields an
// empty store, so the first write is what creates the file. Neither is an
// empty one, which is what a `touch` or an interrupted sync leaves behind
// and holds nothing to lose.
func Load() (*Store, error) {
	path := Path()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Store{SchemaVersion: SchemaVersion}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return &Store{SchemaVersion: SchemaVersion}, nil
	}

	s := &Store{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	if s.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("invalid %s:\n  - schemaVersion: expected %d, got %d", path, SchemaVersion, s.SchemaVersion)
	}
	return s, nil
}

// Save writes the store back atomically, so a reader (or a sync tool
// mid-copy) never observes a half-written file. Projects and items are
// ordered deterministically to keep diffs small when the file is synced
// through git.
func (s *Store) Save() error {
	s.SchemaVersion = SchemaVersion
	slices.SortFunc(s.Projects, func(a, b *Project) int { return strings.Compare(a.ID, b.ID) })
	for _, p := range s.Projects {
		if p.Items == nil {
			p.Items = []Item{}
		}
	}
	if s.Projects == nil {
		s.Projects = []*Project{}
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	path := Path()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".todo-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_, writeErr := tmp.Write(data)
	closeErr := tmp.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		return errors.Join(writeErr, closeErr)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// projectIndex is the one place the id comparison lives. GitHub treats
// owner and repo names case-insensitively, so a remote cloned as
// Dittofleet/Whatagain has to find a project registered as dittofleet/whatagain.
func (s *Store) projectIndex(id string) int {
	return slices.IndexFunc(s.Projects, func(p *Project) bool {
		return strings.EqualFold(p.ID, id)
	})
}

// Project returns the project with the given id, or nil.
func (s *Store) Project(id string) *Project {
	if i := s.projectIndex(id); i >= 0 {
		return s.Projects[i]
	}
	return nil
}

// AddProject registers id and returns the new project. It errors if the
// project already exists.
func (s *Store) AddProject(id string) (*Project, error) {
	if err := ValidateProjectID(id); err != nil {
		return nil, err
	}
	if s.Project(id) != nil {
		return nil, fmt.Errorf("project already exists: %s", id)
	}
	p := &Project{ID: id, Items: []Item{}}
	s.Projects = append(s.Projects, p)
	return p, nil
}

// RemoveProject drops a project and everything in it, returning the
// removed project.
func (s *Store) RemoveProject(id string) (*Project, error) {
	i := s.projectIndex(id)
	if i < 0 {
		return nil, fmt.Errorf("no such project: %s", id)
	}
	p := s.Projects[i]
	s.Projects = slices.Delete(s.Projects, i, i+1)
	return p, nil
}

// FindItemByID looks an item up across every project, since ids are unique
// store-wide. This is what lets `rm <id>` work from any directory.
func (s *Store) FindItemByID(id string) (*Project, int) {
	id = strings.TrimSpace(id)
	for _, p := range s.Projects {
		for i, it := range p.Items {
			if strings.EqualFold(it.ID, id) {
				return p, i
			}
		}
	}
	return nil, -1
}

// newItem builds an item with a fresh id that collides with nothing else
// in the store. Ids stay short because they only ever need to be typed
// once, and widen if a personal-scale store somehow saturates the space.
func (s *Store) newItem(text string) Item {
	for width := 2; ; width++ {
		for attempt := 0; attempt < 16; attempt++ {
			id := randomID(width)
			if p, _ := s.FindItemByID(id); p == nil {
				return Item{
					ID:      id,
					Text:    text,
					Created: time.Now().UTC().Format(time.RFC3339),
				}
			}
		}
	}
}

func randomID(bytes int) string {
	b := make([]byte, bytes)
	// crypto/rand.Read never returns an error on the platforms we build for.
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// AddItem appends text to a project and returns the stored item. It hangs
// off the store because item ids have to be unique store-wide.
func (s *Store) AddItem(p *Project, text string) Item {
	item := s.newItem(text)
	p.Items = append(p.Items, item)
	return item
}

// RemoveItemAt deletes the item at index i and returns it.
func (p *Project) RemoveItemAt(i int) Item {
	item := p.Items[i]
	p.Items = slices.Delete(p.Items, i, i+1)
	return item
}

var projectIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

// ValidateProjectID checks that id looks like a GitHub "owner/name" slug.
func ValidateProjectID(id string) error {
	if !projectIDPattern.MatchString(id) {
		return fmt.Errorf("invalid project: %q is not an owner/name repo id (e.g. dittofleet/whatagain)", id)
	}
	return nil
}
