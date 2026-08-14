package cmd

import (
	"fmt"

	"github.com/sylophi/whatagain/internal/store"
)

const addUsage = `usage: whatagain add [-p <owner/repo>] "<text>"`

// Add stores one item. The note is a single argument, so the shell hands
// it over intact: unquoted text loses apostrophes, globs, and anything
// past an & or a |.
func Add(args []string) error {
	var project string
	rest, err := flags{values: projectFlag(&project)}.parse(args, addUsage)
	if err != nil {
		return err
	}
	if len(rest) > 1 {
		return fmt.Errorf("add takes one note, quoted%s\n%s", quotedSuggestion("add", rest), addUsage)
	}

	text := ""
	if len(rest) == 1 {
		text = normalizeNote(rest[0])
	}
	if text == "" {
		return fmt.Errorf("nothing to add\n%s", addUsage)
	}

	var item store.Item
	var target string
	if err := store.Update(func(s *store.Store) error {
		p, err := resolveProject(s, project)
		if err != nil {
			return err
		}
		item = s.AddItem(p, text)
		target = p.ID
		return nil
	}); err != nil {
		return err
	}

	fmt.Printf("Added %s to %s: %s\n", item.ID, target, item.Text)
	return nil
}
