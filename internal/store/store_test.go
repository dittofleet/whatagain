package store

import "testing"

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
		item := s.AddItem(p, "note")
		if seen[item.ID] {
			t.Fatalf("duplicate id %q at item %d", item.ID, i)
		}
		seen[item.ID] = true
	}
}
