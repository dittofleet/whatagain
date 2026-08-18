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
	"unicode"

	"github.com/dittofleet/whatagain/internal/xdg"
)

// SchemaVersion is what this build writes. Older files load as they are,
// since every version so far has only added optional fields, and are
// stamped with the current version on the next save. A newer file is
// refused: dropping fields this build does not know about would quietly
// delete them from a store synced between machines.
const SchemaVersion = 3

type Item struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	// Description is optional detail that does not fit in a one-line note.
	// It is omitted from the file entirely when empty, so items without one
	// look exactly as they always have.
	Description string `json:"description,omitempty"`
	// Tags are words hung on the item and nothing more. Nothing registers
	// them and no list of them exists anywhere else in the file: a tag is
	// alive exactly as long as an item carries it. They keep the casing and
	// the order they were added in, and compare case-insensitively like ids
	// do, so #CI and #ci are the same word.
	Tags    []string `json:"tags,omitempty"`
	Created string   `json:"created"`
}

// HasTag reports whether the item carries tag.
func (it Item) HasTag(tag string) bool {
	return ContainsTag(it.Tags, tag)
}

// ContainsTag is the one place a tag is compared to a tag, so #CI and #ci
// are the same word everywhere and not just on an item.
func ContainsTag(tags []string, tag string) bool {
	return slices.ContainsFunc(tags, func(t string) bool { return strings.EqualFold(t, tag) })
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
	switch {
	case s.SchemaVersion > SchemaVersion:
		return nil, fmt.Errorf("%s was written by a newer whatagain (schemaVersion %d, this build understands %d)\nUpdate with `whatagain update`", path, s.SchemaVersion, SchemaVersion)
	case s.SchemaVersion < 1:
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

// newID returns an id that collides with nothing else in the store. Ids
// stay short because they only ever need to be typed once, and widen if a
// personal-scale store somehow saturates the space.
func (s *Store) newID() string {
	for width := 2; ; width++ {
		for attempt := 0; attempt < 16; attempt++ {
			id := randomID(width)
			if p, _ := s.FindItemByID(id); p == nil {
				return id
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

// AddItem appends an item to a project, filling in the id and timestamp
// the store owns, and returns what was stored. It hangs off the store
// because item ids have to be unique store-wide. Everything optional about
// an item is a field on the one passed in, so adding another does not
// change this signature.
func (s *Store) AddItem(p *Project, item Item) Item {
	tags := item.Tags
	item.ID, item.Tags = s.newID(), nil
	item.Created = time.Now().UTC().Format(time.RFC3339)
	p.Items = append(p.Items, item)
	// Through AddTags so a new item and an existing one dedupe by the same
	// rule rather than two.
	return p.AddTags(len(p.Items)-1, tags)
}

// SetDescription replaces the description of the item at index i and
// returns it. An empty description leaves the item with none.
func (p *Project) SetDescription(i int, description string) Item {
	p.Items[i].Description = description
	return p.Items[i]
}

// AddTags hangs tags on the item at index i, skipping the ones it already
// carries so a tag cannot land on an item twice.
func (p *Project) AddTags(i int, tags []string) Item {
	it := &p.Items[i]
	for _, tag := range tags {
		if !it.HasTag(tag) {
			it.Tags = append(it.Tags, tag)
		}
	}
	return *it
}

// RemoveTags takes tags back off the item at index i, and reports the ones
// it was not carrying so the caller can refuse the whole command rather
// than half-do it.
func (p *Project) RemoveTags(i int, tags []string) (Item, []string) {
	it := &p.Items[i]
	var missing []string
	for _, tag := range tags {
		if !it.HasTag(tag) {
			missing = append(missing, tag)
		}
	}
	if len(missing) > 0 {
		return *it, missing
	}
	it.Tags = slices.DeleteFunc(it.Tags, func(t string) bool { return ContainsTag(tags, t) })
	if len(it.Tags) == 0 {
		// An empty slice would still marshal as "tags": [], which an item
		// that has none has never had in the file.
		it.Tags = nil
	}
	return *it, nil
}

// ClearTags drops every tag from the item at index i.
func (p *Project) ClearTags(i int) Item {
	p.Items[i].Tags = nil
	return p.Items[i]
}

// RemoveItemAt deletes the item at index i and returns it.
func (p *Project) RemoveItemAt(i int) Item {
	item := p.Items[i]
	p.Items = slices.Delete(p.Items, i, i+1)
	return item
}

var projectIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

// NormalizeTag tidies one tag and refuses what cannot be one. A tag is a
// single word: the leading "#" it is displayed with is optional to type and
// dropped here, and whitespace is an error rather than a silent split into
// two tags.
func NormalizeTag(tag string) (string, error) {
	t := strings.TrimPrefix(strings.TrimSpace(tag), "#")
	if t == "" {
		return "", fmt.Errorf("invalid tag: %q has nothing in it", tag)
	}
	if strings.ContainsFunc(t, unicode.IsSpace) {
		return "", fmt.Errorf("invalid tag: %q is more than one word", tag)
	}
	return t, nil
}

// ValidateProjectID checks that id looks like a GitHub "owner/name" slug.
func ValidateProjectID(id string) error {
	if !projectIDPattern.MatchString(id) {
		return fmt.Errorf("invalid project: %q is not an owner/name repo id (e.g. dittofleet/whatagain)", id)
	}
	return nil
}
