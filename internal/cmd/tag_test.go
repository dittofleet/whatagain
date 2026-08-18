package cmd

import (
	"slices"
	"testing"

	"github.com/dittofleet/whatagain/internal/store"
)

func TestParseTags(t *testing.T) {
	// However they are written, the tags arrive as the same list: split on
	// commas, the displayed # optional, and duplicates dropped in whatever
	// case they were first written.
	got, err := parseTags([]string{"ci,flaky", "#Windows", " ci ", "CI"})
	if err != nil {
		t.Fatalf("parseTags errored: %v", err)
	}
	want := []string{"ci", "flaky", "Windows"}
	if !slices.Equal(got, want) {
		t.Errorf("parseTags = %v, want %v", got, want)
	}

	for _, arg := range []string{"", "#", "two words", ",,"} {
		if _, err := parseTags([]string{arg}); err == nil {
			t.Errorf("parseTags(%q) = nil error, want a rejected tag", arg)
		}
	}
}

func TestTagFlagCollectsEveryValue(t *testing.T) {
	var tags []string
	var project string
	rest, err := flags{values: projectFlag(&project), lists: tagFlag(&tags)}.
		parse([]string{"-t", "ci", "ship it", "--tag=flaky", "--tags", "windows"}, "usage")
	if err != nil {
		t.Fatalf("parse errored: %v", err)
	}
	if len(rest) != 1 || rest[0] != "ship it" {
		t.Errorf("positionals = %v, want just the note", rest)
	}
	if want := []string{"ci", "flaky", "windows"}; !slices.Equal(tags, want) {
		t.Errorf("tags = %v, want %v", tags, want)
	}
}

func TestFilterByTagsWantsAllOfThem(t *testing.T) {
	projects := []*store.Project{{ID: "a/b", Items: []store.Item{
		{ID: "01", Text: "both", Tags: []string{"ci", "flaky"}},
		{ID: "02", Text: "one", Tags: []string{"ci"}},
		{ID: "03", Text: "none"},
	}}}

	filtered := filterByTags(projects, []string{"CI", "flaky"})
	if len(filtered) != 1 || len(filtered[0].Items) != 1 || filtered[0].Items[0].ID != "01" {
		t.Fatalf("filterByTags kept %v, want only the item carrying both", filtered[0].Items)
	}
	// The store keeps everything it loaded, so a second filter still sees
	// the items the first one left out.
	if len(projects[0].Items) != 3 {
		t.Errorf("filtering left the project with %d items, want all 3", len(projects[0].Items))
	}
	if got := filterByTags(projects, nil); len(got[0].Items) != 3 {
		t.Errorf("filtering on no tags kept %d items, want all 3", len(got[0].Items))
	}
}

func TestFlagWillNotEatAnotherFlag(t *testing.T) {
	// `ls -t $TAG --json` with TAG unset used to filter on the word
	// "--json" and report that nothing matched.
	var tags []string
	var asJSON bool
	f := flags{bools: map[string]*bool{"json": &asJSON}, lists: tagFlag(&tags)}
	if _, err := f.parse([]string{"-t", "--json"}, "usage"); err == nil {
		t.Error("parse(-t --json) = nil error, want a missing-value error")
	}
	if asJSON {
		t.Error("--json was consumed as the tag's value")
	}

	// Text that only starts with a dash is still a value, since a
	// description can begin with one.
	var description string
	if _, err := (flags{values: descriptionFlag(&description)}).parse([]string{"-d", "-5 is the bad case"}, "usage"); err != nil {
		t.Fatalf("parse errored: %v", err)
	}
	if description != "-5 is the bad case" {
		t.Errorf("description = %q, want the dash-leading text", description)
	}
}
