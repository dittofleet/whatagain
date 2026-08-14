package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSchemaVersions(t *testing.T) {
	// A store written before descriptions existed still loads, so an
	// update does not strand the file a previous version left behind.
	if _, err := loadFile(t, `{"schemaVersion":1,"projects":[]}`); err != nil {
		t.Errorf("loading a v1 store errored: %v", err)
	}
	// One written by a newer build is refused rather than read and saved
	// back without the fields this build cannot see.
	_, err := loadFile(t, `{"schemaVersion":99,"projects":[]}`)
	if err == nil || !strings.Contains(err.Error(), "whatagain update") {
		t.Errorf("loading a newer store = %v, want an error pointing at `whatagain update`", err)
	}
	if _, err := loadFile(t, `{"projects":[]}`); err == nil {
		t.Error("loading a store with no schemaVersion = nil error, want an invalid-file error")
	}
}

// loadFile writes contents to a store in a temporary config dir and loads it.
func loadFile(t *testing.T, contents string) (*Store, error) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Dir(Path()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return Load()
}

func TestProjectLookupIgnoresCase(t *testing.T) {
	s := &Store{Projects: []*Project{{ID: "dittofleet/whatagain"}}}

	// A remote cloned as Dittofleet/Whatagain names the same GitHub repo.
	p := s.Project("Dittofleet/Whatagain")
	if p == nil {
		t.Fatal("Project(\"Dittofleet/Whatagain\") = nil, want the registered project")
	}
	if p.ID != "dittofleet/whatagain" {
		t.Errorf("p.ID = %q, want the id as registered", p.ID)
	}
	if _, err := s.AddProject("DITTOFLEET/WHATAGAIN"); err == nil {
		t.Error("AddProject with different casing = nil error, want a duplicate error")
	}
	if _, err := s.RemoveProject("DITTOFLEET/WHATAGAIN"); err != nil {
		t.Errorf("RemoveProject with different casing errored: %v", err)
	}
}

func TestNewItemIDsAreUnique(t *testing.T) {
	s := &Store{}
	p, err := s.AddProject("a/b")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for i := 0; i < 500; i++ {
		item := s.AddItem(p, "note", "")
		if seen[item.ID] {
			t.Fatalf("duplicate id %q at item %d", item.ID, i)
		}
		seen[item.ID] = true
	}
}
