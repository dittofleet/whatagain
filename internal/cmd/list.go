package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dittofleet/whatagain/internal/store"
)

const listUsage = "usage: whatagain ls [-p <owner/repo>] [--all] [--json]"

// List prints items. With no flags it shows the current repo's project,
// falling back to every project when the working directory does not
// belong to one, which is what makes a bare `whatagain ls` useful anywhere.
func List(args []string) error {
	var project string
	var all, asJSON bool
	f := flags{
		bools:  map[string]*bool{"all": &all, "a": &all, "json": &asJSON},
		values: projectFlag(&project),
	}
	rest, err := f.parse(args, listUsage)
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

	if asJSON {
		return writeJSON(struct {
			Projects []*store.Project `json:"projects"`
		}{emptyToSlice(shown)})
	}
	printItems(shown, scoped)
	return nil
}

// printItems renders the listing. scoped means projects holds the single
// project that was asked for, which is the only case where an empty
// project is worth naming.
func printItems(projects []*store.Project, scoped bool) {
	if itemCount(projects) == 0 {
		if scoped {
			fmt.Printf("%s has no items.\n", projects[0].ID)
			return
		}
		fmt.Println("No items.")
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
			fmt.Printf("  %s  %s\n", it.ID, it.Text)
		}
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
