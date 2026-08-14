package cmd

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/dittofleet/whatagain/internal/store"
)

const removeUsage = `usage: whatagain rm [-p <owner/repo>] <id>...
       whatagain rm [-p <owner/repo>] "<text>"`

// Remove deletes items addressed either by id or by their text. Ids can be
// given several at a time. Text is one quoted argument, matched within a
// single project.
func Remove(args []string) error {
	var project string
	rest, err := flags{values: projectFlag(&project)}.parse(args, removeUsage)
	if err != nil {
		return err
	}
	// A blank argument is almost always an unset variable, and matching it
	// as text would substring-match every item in the project.
	if len(rest) == 0 || slices.ContainsFunc(rest, func(a string) bool { return normalizeNote(a) == "" }) {
		return fmt.Errorf("nothing to remove\n%s", removeUsage)
	}

	var removed []removal
	err = store.Update(func(s *store.Store) error {
		// Ids are unique store-wide, so removing by id needs no project
		// context and works from any directory.
		hits, unresolved := resolveIDs(s, rest)
		switch {
		case len(unresolved) == 0:
			removed = removeHits(hits)
			return nil
		case len(hits) > 0:
			// Some arguments were ids, so this is a mistyped id rather
			// than an unquoted note. Implies more than one argument: a
			// lone unresolved one could not have produced a hit.
			return fmt.Errorf("no item with id: %s", strings.Join(unresolved, ", "))
		case len(rest) > 1:
			return fmt.Errorf("rm takes item ids, or one quoted note%s\n%s", quotedSuggestion("rm", rest), removeUsage)
		}

		p, err := resolveProject(s, project)
		if err != nil {
			return err
		}
		i, err := matchItem(p, normalizeNote(rest[0]))
		if err != nil {
			return err
		}
		removed = []removal{{p.ID, p.RemoveItemAt(i)}}
		return nil
	})
	if err != nil {
		return err
	}

	for _, r := range removed {
		fmt.Printf("Removed %s from %s: %s\n", r.item.ID, r.project, r.item.Text)
	}
	return nil
}

type removal struct {
	project string
	item    store.Item
}

type hit struct {
	project *store.Project
	index   int
}

// resolveIDs locates every argument that names an item, without touching
// the store, and reports the ones that name nothing.
func resolveIDs(s *store.Store, args []string) (hits []hit, unresolved []string) {
	seen := make(map[string]bool, len(args))
	for _, arg := range args {
		p, i := s.FindItemByID(arg)
		if p == nil {
			unresolved = append(unresolved, arg)
			continue
		}
		if seen[p.Items[i].ID] {
			continue
		}
		seen[p.Items[i].ID] = true
		hits = append(hits, hit{p, i})
	}
	return hits, unresolved
}

// removeHits deletes located items, highest index first so each removal
// leaves the rest valid.
func removeHits(hits []hit) []removal {
	slices.SortFunc(hits, func(a, b hit) int { return cmp.Compare(b.index, a.index) })
	removed := make([]removal, 0, len(hits))
	for _, h := range hits {
		removed = append(removed, removal{h.project.ID, h.project.RemoveItemAt(h.index)})
	}
	return removed
}

// matchItem finds the one item in p whose text identifies it, preferring a
// whole-text match over a substring one. Ambiguity is an error listing the
// candidates by id rather than a guess.
func matchItem(p *store.Project, text string) (int, error) {
	// Guarded because empty text is a substring of every item, which would
	// make "the only match" mean "the first item".
	if text == "" {
		return 0, fmt.Errorf("no text to match against %s", p.ID)
	}
	needle := strings.ToLower(text)

	var exact, partial []int
	for i, it := range p.Items {
		got := strings.ToLower(it.Text)
		switch {
		case got == needle:
			exact = append(exact, i)
		case strings.Contains(got, needle):
			partial = append(partial, i)
		}
	}

	matches := exact
	if len(matches) == 0 {
		matches = partial
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return 0, fmt.Errorf("no item in %s matches %q", p.ID, text)
	default:
		var b strings.Builder
		fmt.Fprintf(&b, "%q matches %d items in %s:\n", text, len(matches), p.ID)
		for _, i := range matches {
			fmt.Fprintf(&b, "  %s  %s\n", p.Items[i].ID, p.Items[i].Text)
		}
		b.WriteString("Remove it by id instead")
		return 0, fmt.Errorf("%s", b.String())
	}
}
