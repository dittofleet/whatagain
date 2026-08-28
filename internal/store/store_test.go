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
		// Half the items land on the global list, which shares the id
		// space with every project.
		target := p
		if i%2 == 0 {
			target = s.Global()
		}
		item := s.AddItem(target, Item{Text: "note"})
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

func TestGlobalListRoundTrips(t *testing.T) {
	s, err := loadFile(t, `{"schemaVersion":4,"items":[{"id":"01","text":"renew the passport","created":"2026-01-01T00:00:00Z"}],"projects":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	if items := s.Global().Items; len(items) != 1 || items[0].Text != "renew the passport" {
		t.Fatalf("global items = %v, want the one from the file", items)
	}
	// Ids are unique store-wide, so a global item is found like any other,
	// which is what keeps desc, tag, and rm working on it with no flag.
	if p, i := s.FindItemByID("01"); p == nil || p.ID != "" || i != 0 {
		t.Errorf("FindItemByID(\"01\") = %v, %d; want the global list", p, i)
	}

	// The projects are empty here, so the only "items" key the file can
	// hold is the global list's.
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"items"`) {
		t.Errorf("saved store has no top-level items array:\n%s", data)
	}

	// An emptied list leaves the file without the field, exactly as a
	// store that never had one looks.
	s.Global().RemoveItemAt(0)
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	if data, err = os.ReadFile(Path()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"items"`) {
		t.Errorf("saved store still has an items array after the last removal:\n%s", data)
	}
}

func TestLoadWithoutGlobalList(t *testing.T) {
	// A store from before the global list existed, whose only "items" keys
	// are the nested ones projects have always had. Loading it must not
	// invent a global list out of them.
	s, err := loadFile(t, `{"schemaVersion":3,"projects":[{"id":"a/b","items":[{"id":"01","text":"note","created":"2026-01-01T00:00:00Z"}]}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if items := s.Global().Items; len(items) != 0 {
		t.Errorf("global items = %v, want none", items)
	}
	if len(s.Projects) != 1 || s.Projects[0].ID != "a/b" || len(s.Projects[0].Items) != 1 {
		t.Errorf("projects = %v, want a/b with its one item", s.Projects)
	}
}
