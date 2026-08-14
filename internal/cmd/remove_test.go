package cmd

import (
	"strings"
	"testing"

	"github.com/sylophi/whatagain/internal/store"
)

func project(texts ...string) *store.Project {
	p := &store.Project{ID: "a/b"}
	for i, t := range texts {
		p.Items = append(p.Items, store.Item{ID: string(rune('a' + i)), Text: t})
	}
	return p
}

func TestMatchItem(t *testing.T) {
	p := project("fix the flaky land test", "fix the docs", "SHIP the windows build")

	cases := map[string]int{
		"fix the docs":            1, // whole text
		"FIX THE DOCS":            1, // case-insensitive
		"windows":                 2, // substring
		"ship the windows build":  2,
		"fix the flaky land test": 0,
	}
	for text, want := range cases {
		got, err := matchItem(p, text)
		if err != nil {
			t.Errorf("matchItem(%q) errored: %v", text, err)
			continue
		}
		if got != want {
			t.Errorf("matchItem(%q) = %d, want %d", text, got, want)
		}
	}

	// "fix" hits two items, and guessing between them would delete the
	// wrong note.
	if _, err := matchItem(p, "fix"); err == nil || !strings.Contains(err.Error(), "matches 2 items") {
		t.Errorf("matchItem(\"fix\") error = %v, want an ambiguity error", err)
	}
	if _, err := matchItem(p, "nope"); err == nil {
		t.Error("matchItem(\"nope\") = nil error, want a not-found error")
	}

	// An exact match wins over a substring one, so an item whose text is a
	// prefix of another stays removable.
	p2 := project("deploy", "deploy the worker")
	if got, err := matchItem(p2, "deploy"); err != nil || got != 0 {
		t.Errorf("matchItem(\"deploy\") = %d, %v; want 0, nil", got, err)
	}
}

func TestResolveAndRemoveIDs(t *testing.T) {
	s := &store.Store{Projects: []*store.Project{
		{ID: "a/b", Items: []store.Item{{ID: "1a", Text: "one"}, {ID: "2b", Text: "two"}, {ID: "3c", Text: "three"}}},
		{ID: "c/d", Items: []store.Item{{ID: "4d", Text: "four"}}},
	}}

	hits, unresolved := resolveIDs(s, []string{"1a", "3c", "4d"})
	if len(unresolved) != 0 {
		t.Fatalf("unresolved = %v, want none", unresolved)
	}
	// Removing several at once must not shift the remaining indices out
	// from under the later removals.
	if removed := removeHits(hits); len(removed) != 3 {
		t.Fatalf("removeHits = %v, want 3 removals", removed)
	}
	if len(s.Projects[0].Items) != 1 || s.Projects[0].Items[0].ID != "2b" {
		t.Errorf("remaining items = %v, want just 2b", s.Projects[0].Items)
	}
	if len(s.Projects[1].Items) != 0 {
		t.Errorf("c/d items = %v, want none", s.Projects[1].Items)
	}

	// Resolving must not mutate: an unknown id has to leave the store
	// alone so the command can fail cleanly.
	s2 := &store.Store{Projects: []*store.Project{
		{ID: "a/b", Items: []store.Item{{ID: "1a", Text: "one"}}},
	}}
	hits, unresolved = resolveIDs(s2, []string{"1a", "zz"})
	if len(hits) != 1 || len(unresolved) != 1 || unresolved[0] != "zz" {
		t.Errorf("resolveIDs = %v, %v; want one hit and zz unresolved", hits, unresolved)
	}
	if len(s2.Projects[0].Items) != 1 {
		t.Error("resolveIDs mutated the store")
	}
}

func TestQuotedSuggestion(t *testing.T) {
	got := quotedSuggestion("add", []string{"fix", "the", "docs"})
	if want := "\n  whatagain add \"fix the docs\""; got != want {
		t.Errorf("quotedSuggestion = %q, want %q", got, want)
	}
	// A note already containing a quote would render a broken suggestion.
	if got := quotedSuggestion("add", []string{`say "hi"`}); got != "" {
		t.Errorf("quotedSuggestion with a quote = %q, want empty", got)
	}
}
