package store

import (
	"os"
	"path/filepath"
	"slices"
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
		item := s.AddItem(p, Item{Text: "note"})
		if seen[item.ID] {
			t.Fatalf("duplicate id %q at item %d", item.ID, i)
		}
		seen[item.ID] = true
	}
}

func TestTagsIgnoreCase(t *testing.T) {
	p := &Project{ID: "a/b", Items: []Item{{ID: "01", Text: "note"}}}

	// A tag the item already carries does not land twice, whatever case it
	// is written in, and the first spelling is the one kept.
	item := p.AddTags(0, []string{"CI", "ci", "flaky"})
	if want := []string{"CI", "flaky"}; !slices.Equal(item.Tags, want) {
		t.Errorf("tags = %v, want %v", item.Tags, want)
	}
	if !item.HasTag("ci") {
		t.Error("HasTag(\"ci\") = false, want true for an item tagged CI")
	}

	// A tag the item is not carrying stops the whole removal, so a typo
	// cannot silently take half the tags off.
	if _, missing := p.RemoveTags(0, []string{"flaky", "windows"}); len(missing) != 1 || missing[0] != "windows" {
		t.Errorf("missing = %v, want just the tag the item lacks", missing)
	}
	if len(p.Items[0].Tags) != 2 {
		t.Errorf("a refused removal left %v, want both tags", p.Items[0].Tags)
	}

	item, _ = p.RemoveTags(0, []string{"cI", "FLAKY"})
	if item.Tags != nil {
		t.Errorf("tags = %v, want nil so the item marshals without them", item.Tags)
	}
}

func TestNormalizeTag(t *testing.T) {
	cases := map[string]string{"ci": "ci", "  #Flaky ": "Flaky", "needs/design": "needs/design"}
	for in, want := range cases {
		got, err := NormalizeTag(in)
		if err != nil {
			t.Errorf("NormalizeTag(%q) errored: %v", in, err)
		} else if got != want {
			t.Errorf("NormalizeTag(%q) = %q, want %q", in, got, want)
		}
	}
	for _, in := range []string{"", "   ", "#", "two words", "line\nbreak"} {
		if _, err := NormalizeTag(in); err == nil {
			t.Errorf("NormalizeTag(%q) = nil error, want a rejected tag", in)
		}
	}
}
