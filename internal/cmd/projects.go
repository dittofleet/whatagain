package cmd

import (
	"fmt"

	"github.com/sylophi/whatagain/internal/repo"
	"github.com/sylophi/whatagain/internal/store"
)

const projectsUsage = `usage: whatagain projects [--json]
       whatagain projects add [<owner/repo>]
       whatagain projects rm <owner/repo>... [--yes]`

// Projects dispatches the project subcommands, defaulting to the listing.
func Projects(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "add":
			return projectAdd(args[1:])
		case "rm", "remove":
			return projectRemove(args[1:])
		}
	}
	return projectList(args)
}

func projectList(args []string) error {
	var asJSON bool
	rest, err := flags{bools: map[string]*bool{"json": &asJSON}}.parse(args, projectsUsage)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("unknown project command: %s\n%s", rest[0], projectsUsage)
	}

	s, err := store.Load()
	if err != nil {
		return err
	}
	current := currentProject(s)

	if asJSON {
		type entry struct {
			ID      string `json:"id"`
			Items   int    `json:"items"`
			Current bool   `json:"current"`
		}
		entries := []entry{}
		for _, p := range s.Projects {
			entries = append(entries, entry{p.ID, len(p.Items), p == current})
		}
		return writeJSON(struct {
			Projects []entry `json:"projects"`
		}{entries})
	}

	if len(s.Projects) == 0 {
		fmt.Println("No projects yet. Register one with `whatagain projects add`.")
		return nil
	}

	width := 0
	for _, p := range s.Projects {
		width = max(width, len(p.ID))
	}
	for _, p := range s.Projects {
		marker := " "
		if p == current {
			marker = "*"
		}
		fmt.Printf("%s %-*s  %s\n", marker, width, p.ID, plural(len(p.Items), "item"))
	}
	return nil
}

func projectAdd(args []string) error {
	rest, err := flags{}.parse(args, projectsUsage)
	if err != nil {
		return err
	}
	if len(rest) > 1 {
		return fmt.Errorf("expected at most one project\n%s", projectsUsage)
	}

	id := ""
	if len(rest) == 1 {
		id = rest[0]
	} else if id, err = repo.Current(); err != nil {
		return fmt.Errorf("%w\nName the project explicitly: `whatagain projects add <owner/repo>`", err)
	}

	if err := store.Update(func(s *store.Store) error {
		_, err := s.AddProject(id)
		return err
	}); err != nil {
		return err
	}
	fmt.Printf("Added project %s\n", id)
	return nil
}

func projectRemove(args []string) error {
	var yes bool
	rest, err := flags{bools: yesFlag(&yes)}.parse(args, projectsUsage)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return fmt.Errorf("no project given\n%s", projectsUsage)
	}

	// One pass is enough: Update saves nothing if this returns an error,
	// so a typo in the second argument cannot half-apply the command.
	removed := make([]*store.Project, 0, len(rest))
	if err := store.Update(func(s *store.Store) error {
		removed = removed[:0]
		for _, id := range rest {
			p := s.Project(id)
			if p == nil {
				return fmt.Errorf("no such project: %s", id)
			}
			if len(p.Items) > 0 && !yes {
				return fmt.Errorf("%s still has %s\nRemove them first, or pass --yes to drop the project and its items", p.ID, plural(len(p.Items), "item"))
			}
			if _, err := s.RemoveProject(id); err != nil {
				return err
			}
			removed = append(removed, p)
		}
		return nil
	}); err != nil {
		return err
	}

	for _, p := range removed {
		fmt.Printf("Removed project %s (%s)\n", p.ID, plural(len(p.Items), "item"))
	}
	return nil
}

// plural takes a singular noun with a regular plural, which covers the
// two it is ever called with.
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
