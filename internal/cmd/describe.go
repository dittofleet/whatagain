package cmd

import (
	"fmt"

	"github.com/dittofleet/whatagain/internal/store"
)

const describeUsage = `usage: whatagain desc <id> "<text>"
       whatagain desc <id> --clear`

// Describe attaches detail to an item that already exists, or drops the
// detail it has. Items are addressed by id, which is unique store-wide, so
// this works from any directory and needs no project.
func Describe(args []string) error {
	var clear bool
	rest, err := flags{bools: map[string]*bool{"clear": &clear}}.parse(args, describeUsage)
	if err != nil {
		return err
	}

	switch {
	case len(rest) == 0:
		return fmt.Errorf("nothing to describe\n%s", describeUsage)
	case clear && len(rest) > 1:
		return fmt.Errorf("--clear takes no text\n%s", describeUsage)
	case !clear && len(rest) == 1:
		return fmt.Errorf("no description given\n%s", describeUsage)
	}
	id := rest[0]

	var item store.Item
	var target string
	if err := store.Update(func(s *store.Store) error {
		// The id is resolved before the arguments are judged, so text the
		// shell has split apart is reported as the unquoted description it
		// is only once we know the command was aimed at a real item.
		p, i := s.FindItemByID(id)
		if p == nil {
			return fmt.Errorf("no item with id: %s", id)
		}
		if len(rest) > 2 {
			// Storing only the first word would look like it worked.
			return fmt.Errorf("desc takes one description, quoted%s\n%s", quotedSuggestion("desc "+id, rest[1:]), describeUsage)
		}

		description := ""
		if !clear {
			description = normalizeDescription(rest[1])
			if description == "" {
				return fmt.Errorf("nothing to describe %s with\nClear it with `whatagain desc %s --clear`", id, id)
			}
		}
		item, target = p.SetDescription(i, description), p.ID
		return nil
	}); err != nil {
		return err
	}

	if clear {
		fmt.Printf("Cleared the description of %s in %s\n", item.ID, target)
		return nil
	}
	fmt.Printf("Described %s in %s: %s\n", item.ID, target, item.Text)
	printDescription(2, item.Description)
	return nil
}
