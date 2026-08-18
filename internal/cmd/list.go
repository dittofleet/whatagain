package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dittofleet/whatagain/internal/store"
)

const listUsage = "usage: whatagain ls [-p <owner/repo>] [-t <tag>...] [--all] [--json]"

// List prints items. With no flags it shows the current repo's project,
// falling back to every project when the working directory does not
// belong to one, which is what makes a bare `whatagain ls` useful anywhere.
func List(args []string) error {
	var project string
	var tagArgs []string
	var all, asJSON bool
	f := flags{
		bools:  map[string]*bool{"all": &all, "a": &all, "json": &asJSON},
		values: projectFlag(&project),
		lists:  tagFlag(&tagArgs),
	}
	rest, err := f.parse(args, listUsage)
	if err != nil {
		return err
	}
	tags, err := parseTags(tagArgs)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("unexpected arguments: %v\n%s", rest, listUsage)
	}
	if all && project != "" {
		return fmt.Errorf("--all and --project are mutually exclusive\n%s", listUsage)
	}

	s, err := store.Load()
	if err != nil {
		return err
	}

	// Showing everything is the fallback, so only the two arms that narrow
	// to a single project have to say anything.
	shown, scoped := s.Projects, false
	switch {
	case project != "":
		p, err := resolveProject(s, project)
		if err != nil {
			return err
		}
		shown, scoped = []*store.Project{p}, true
	case !all:
		if p := currentProject(s); p != nil {
			shown, scoped = []*store.Project{p}, true
		}
	}

	shown = filterByTags(shown, tags)
	if asJSON {
		return writeJSON(struct {
			Projects []*store.Project `json:"projects"`
		}{emptyToSlice(shown)})
	}
	printItems(shown, scoped, tags)
	return nil
}

// filterByTags narrows every project to the items carrying all of tags,
// which is what makes a filter worth having once the list is long: each
// tag you add takes items away. The projects are copies, so the store in
// memory keeps everything it loaded.
func filterByTags(projects []*store.Project, tags []string) []*store.Project {
	if len(tags) == 0 {
		return projects
	}
	filtered := make([]*store.Project, 0, len(projects))
	for _, p := range projects {
		kept := []store.Item{}
		for _, it := range p.Items {
			if hasEveryTag(it, tags) {
				kept = append(kept, it)
			}
		}
		// A copy of the project rather than a new one, so a field added to
		// it later cannot go missing from a filtered listing.
		narrowed := *p
		narrowed.Items = kept
		filtered = append(filtered, &narrowed)
	}
	return filtered
}

func hasEveryTag(it store.Item, tags []string) bool {
	for _, tag := range tags {
		if !it.HasTag(tag) {
			return false
		}
	}
	return true
}

// printItems renders the listing. scoped means projects holds the single
// project that was asked for, which is the only case where an empty
// project is worth naming. tags are the ones filtered on, so nothing left
// reads as a filter that matched rather than an empty list.
func printItems(projects []*store.Project, scoped bool, tags []string) {
	if itemCount(projects) == 0 {
		where := ""
		if scoped {
			where = " in " + projects[0].ID
		}
		switch {
		case len(tags) > 0:
			fmt.Printf("No items tagged %s%s.\n", formatTags(tags), where)
		case scoped:
			fmt.Printf("%s has no items.\n", projects[0].ID)
		default:
			fmt.Println("No items.")
		}
		return
	}

	first := true
	for _, p := range projects {
		if len(p.Items) == 0 {
			continue
		}
		if !first {
			fmt.Println()
		}
		first = false
		fmt.Println(p.ID)
		for _, it := range p.Items {
			// Tags ride on the note's own line, so an item still reads as
			// one line unless it has detail hanging under it.
			fmt.Printf("  %s  %s%s\n", it.ID, it.Text, tagSuffix(it.Tags))
			// Detail hangs under the note, aligned with it, so an item that
			// has none still reads as the single line it always was.
			printDescription(4+len(it.ID), it.Description)
		}
	}
}

// printDescription writes each line of a description indented by width
// spaces, and nothing at all when there is none.
func printDescription(width int, description string) {
	if description == "" {
		return
	}
	indent := strings.Repeat(" ", width)
	// A blank line inside a description keeps the indent, so it cannot be
	// mistaken for the empty line that separates one project from the next.
	for _, line := range strings.Split(description, "\n") {
		fmt.Println(indent + line)
	}
}

// emptyToSlice keeps JSON output as [] rather than null when nothing matches.
func emptyToSlice(p []*store.Project) []*store.Project {
	if p == nil {
		return []*store.Project{}
	}
	return p
}

func itemCount(projects []*store.Project) int {
	n := 0
	for _, p := range projects {
		n += len(p.Items)
	}
	return n
}

func writeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
