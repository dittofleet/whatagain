package cmd

import (
	"slices"
	"testing"

	"github.com/dittofleet/whatagain/internal/store"
)

func TestAdoptTakesOnlyWhatIsNew(t *testing.T) {
	s := &store.Store{Projects: []*store.Project{
		{ID: "dittofleet/whatagain", Items: []store.Item{{ID: "01", Text: "ship it"}}},
	}}

	// A repo the store already holds, one in the casing GitHub does not
	// distinguish, and one it has never seen.
	added := adopt(s, []string{"dittofleet/whatagain", "Dittofleet/Whatagain", "sylophi/terrier"})
	if !slices.Equal(added, []string{"sylophi/terrier"}) {
		t.Errorf("adopt added %v, want only the repo the store did not have", added)
	}
	if len(s.Projects) != 2 {
		t.Fatalf("store holds %d projects, want 2", len(s.Projects))
	}
	if got := s.Project("sylophi/terrier"); got == nil || len(got.Items) != 0 {
		t.Errorf("the adopted project = %v, want one with no items", got)
	}
	if got := s.Project("dittofleet/whatagain"); got == nil || len(got.Items) != 1 {
		t.Errorf("the project that was already there = %v, want its one item untouched", got)
	}

	// Adopting the same registry again is what every later command does.
	if again := adopt(s, []string{"dittofleet/whatagain", "sylophi/terrier"}); len(again) != 0 {
		t.Errorf("adopting twice added %v, want nothing the second time", again)
	}
}

func TestAdoptSkipsWhatCouldNotBeTypedIn(t *testing.T) {
	// Terrier names a repo by its GitHub origin, which is an owner/name
	// slug. Anything else is passed over rather than made a project
	// whatagain would have refused to be given directly.
	s := &store.Store{}
	added := adopt(s, []string{"not-a-slug", "", "a/b/c", "dittofleet/whatagain"})
	if !slices.Equal(added, []string{"dittofleet/whatagain"}) {
		t.Errorf("adopt added %v, want only the owner/name one", added)
	}
	if len(s.Projects) != 1 {
		t.Errorf("store holds %d projects, want just the one", len(s.Projects))
	}
}

func TestAdoptWithoutTerrierChangesNothing(t *testing.T) {
	// What every command does on a machine with no terrier installed,
	// where Slugs reports nothing. The store has to come through it
	// exactly as it was, or whatagain would be rewriting a synced file for
	// a registry that is not there.
	s := &store.Store{Projects: []*store.Project{
		{ID: "dittofleet/whatagain", Items: []store.Item{{ID: "01", Text: "ship it"}}},
	}}
	if added := adopt(s, nil); len(added) != 0 {
		t.Errorf("adopt added %v, want nothing to adopt", added)
	}
	if len(s.Projects) != 1 || len(s.Projects[0].Items) != 1 {
		t.Errorf("the store changed with no terrier to adopt from: %v", s.Projects)
	}
}
