package cmd

import (
	"fmt"
	"strings"

	"github.com/dittofleet/whatagain/internal/store"
)

const tagUsage = `usage: whatagain tag <id> <tag>...
       whatagain tag <id> --clear`

const untagUsage = `usage: whatagain untag <id> <tag>...`

// Tag hangs words on an item that already exists, or takes them all off.
// Like desc, it addresses the item by id, which is unique store-wide, so it
// works from any directory and needs no project.
func Tag(args []string) error {
	var clear bool
	rest, err := flags{bools: map[string]*bool{"clear": &clear}}.parse(args, tagUsage)
	if err != nil {
		return err
	}
	switch {
	case len(rest) == 0:
		return fmt.Errorf("nothing to tag\n%s", tagUsage)
	case clear && len(rest) > 1:
		return fmt.Errorf("--clear takes no tags\n%s", tagUsage)
	case !clear && len(rest) == 1:
		// An unquoted #ci never reaches us: the shell reads it as the start
		// of a comment and drops the rest of the line.
		return fmt.Errorf("no tags to add to %s\nTags are bare words, so write `whatagain tag %s ci`, or quote the #\n%s", rest[0], rest[0], tagUsage)
	}
	id := rest[0]

	tags, err := parseTags(rest[1:])
	if err != nil {
		return err
	}

	item, target, err := updateItem(id, func(p *store.Project, i int) (store.Item, error) {
		if clear {
			return p.ClearTags(i), nil
		}
		return p.AddTags(i, tags), nil
	})
	if err != nil {
		return err
	}

	if clear {
		fmt.Printf("Cleared the tags of %s in %s\n", item.ID, target)
		return nil
	}
	fmt.Printf("Tagged %s in %s: %s%s\n", item.ID, target, item.Text, tagSuffix(item.Tags))
	return nil
}

// Untag takes named tags back off an item. A tag the item is not carrying
// stops the whole command, so nothing half-happens on a typo.
func Untag(args []string) error {
	rest, err := flags{}.parse(args, untagUsage)
	if err != nil {
		return err
	}
	switch {
	case len(rest) == 0:
		return fmt.Errorf("nothing to untag\n%s", untagUsage)
	case len(rest) == 1:
		return fmt.Errorf("no tags to take off %s\nDrop them all with `whatagain tag %s --clear`\n%s", rest[0], rest[0], untagUsage)
	}
	id := rest[0]

	tags, err := parseTags(rest[1:])
	if err != nil {
		return err
	}

	item, target, err := updateItem(id, func(p *store.Project, i int) (store.Item, error) {
		updated, missing := p.RemoveTags(i, tags)
		if len(missing) > 0 {
			return store.Item{}, fmt.Errorf("%s is not tagged %s", id, formatTags(missing))
		}
		return updated, nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("Untagged %s in %s: %s%s\n", item.ID, target, item.Text, tagSuffix(item.Tags))
	return nil
}

// parseTags turns however the tags were written into the list to act on.
// Each argument may itself be a comma-separated run, so -t ci,flaky and
// -t ci -t flaky and a bare `tag 2437 ci flaky` all arrive the same way.
// Duplicates are dropped, keeping the first spelling of each word.
func parseTags(args []string) ([]string, error) {
	tags := []string{}
	for _, arg := range args {
		for _, raw := range strings.Split(arg, ",") {
			tag, err := store.NormalizeTag(raw)
			if err != nil {
				return nil, err
			}
			if !store.ContainsTag(tags, tag) {
				tags = append(tags, tag)
			}
		}
	}
	return tags, nil
}

// formatTags renders tags the way they are displayed, with the # that is
// optional to type.
func formatTags(tags []string) string {
	shown := make([]string, 0, len(tags))
	for _, tag := range tags {
		shown = append(shown, "#"+tag)
	}
	return strings.Join(shown, " ")
}

// tagSuffix is formatTags for the end of a line about an item, and nothing
// at all when the item carries none.
func tagSuffix(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return "  " + formatTags(tags)
}
