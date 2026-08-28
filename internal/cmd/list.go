package cmd

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/dittofleet/whatagain/internal/store"
)

const listUsage = "usage: whatagain ls [-p <owner/repo> | -g] [-t <tag>...] [--all] [--json]"

// List prints items. With no flags it shows the current repo's project,
// falling back to everything, the global list included, when the working
// directory does not belong to one, which is what makes a bare
// `whatagain ls` useful anywhere.
func List(args []string) error {
	var project string
	var tagArgs []string
	var all, global, asJSON bool
	bools := map[string]*bool{"all": &all, "a": &all, "json": &asJSON}
	maps.Copy(bools, globalFlag(&global))
	f := flags{
		bools:  bools,
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
	if global && project != "" {
		return fmt.Errorf("--global and --project are mutually exclusive\n%s", listUsage)
	}
	if global && all {
		return fmt.Errorf("--global and --all are mutually exclusive\n%s", listUsage)
	}

	s, err := openStore()
	if err != nil {
		return err
	}

	// Showing everything is the fallback, so only the arms that narrow to
	// a single list have to say anything.
	shown, scoped := s.Lists(), false
	switch {
	case global:
		shown, scoped = []*store.Project{s.Global()}, true
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
		// The global list keeps its shape in the output too: items at the
		// top level, not a project with a blank id.
		items, projects := []store.Item{}, []*store.Project{}
		for _, p := range shown {
			if p.ID == "" {
				items = append(items, p.Items...)
			} else {
				projects = append(projects, p)
			}
		}
		return writeJSON(struct {
			Projects []*store.Project `json:"projects"`
			Items    []store.Item     `json:"items"`
		}{projects, items})
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
// list that was asked for, which is the only case where an empty one is
// worth naming. tags are the ones filtered on, so nothing left
// reads as a filter that matched rather than an empty list.
func printItems(projects []*store.Project, scoped bool, tags []string) {
	if itemCount(projects) == 0 {
		where := ""
		if scoped {
			where = " in " + listName(projects[0].ID)
		}
		switch {
		case len(tags) > 0:
			fmt.Printf("No items tagged %s%s.\n", formatTags(tags), where)
		case scoped && projects[0].ID == "":
			fmt.Println("The global list has no items.")
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
		fmt.Println(listHeader(p.ID))
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

// listHeader is the line a listing opens a list with. The global list
// gets a name no project id can be mistaken for, since every one of
// those has a slash in it.
func listHeader(id string) string {
	if id == "" {
		return "(no project)"
	}
	return id
}

// listName is what prose calls the list an item lives in: the project
// id, or "the global list" for the one that has none.
func listName(id string) string {
	if id == "" {
		return "the global list"
	}
	return id
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
