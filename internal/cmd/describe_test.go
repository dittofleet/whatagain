package cmd

import (
	"maps"
	"testing"
)

func TestNormalizeDescription(t *testing.T) {
	cases := map[string]string{
		"":                               "",
		"   \n\t\n  ":                    "",
		"  only fails in CI  ":           "  only fails in CI",
		"one\r\ntwo":                     "one\ntwo",
		"\n\nkeeps\n\nthe gap\n\n":       "keeps\n\nthe gap",
		"trailing space   \nand a tab\t": "trailing space\nand a tab",
	}
	for in, want := range cases {
		if got := normalizeDescription(in); got != want {
			t.Errorf("normalizeDescription(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDescriptionFlagParses(t *testing.T) {
	var project, description string
	values := projectFlag(&project)
	maps.Copy(values, descriptionFlag(&description))
	f := flags{values: values}

	rest, err := f.parse([]string{"-d", "why it matters", "ship it", "-p", "a/b"}, "usage")
	if err != nil {
		t.Fatalf("parse errored: %v", err)
	}
	if len(rest) != 1 || rest[0] != "ship it" {
		t.Errorf("positionals = %v, want just the note", rest)
	}
	if description != "why it matters" || project != "a/b" {
		t.Errorf("description = %q, project = %q; want both set", description, project)
	}

	// Merging the two sets must not lose either one's aliases.
	for _, alias := range []string{"--desc", "--description"} {
		description = ""
		if _, err := f.parse([]string{alias, "detail"}, "usage"); err != nil {
			t.Fatalf("parse(%s) errored: %v", alias, err)
		}
		if description != "detail" {
			t.Errorf("%s did not set the description", alias)
		}
	}
}
