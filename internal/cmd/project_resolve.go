package cmd

import (
	"errors"
	"fmt"

	"github.com/dittofleet/whatagain/internal/repo"
	"github.com/dittofleet/whatagain/internal/store"
)

// resolveProject picks the project a command acts on: the --project value
// when given, otherwise the repo the working directory belongs to.
// Registration is always explicit, so an unregistered repo is an error
// carrying the command that would fix it.
func resolveProject(s *store.Store, override string) (*store.Project, error) {
	if override != "" {
		if err := store.ValidateProjectID(override); err != nil {
			return nil, err
		}
		p := s.Project(override)
		if p == nil {
			return nil, fmt.Errorf("no such project: %s\nRegister it with `whatagain projects add %s`", override, override)
		}
		return p, nil
	}

	id, err := repo.Current()
	if err != nil {
		if errors.Is(err, repo.ErrNoProject) {
			return nil, fmt.Errorf("%w\nPick a project with `--project <owner/repo>`", err)
		}
		return nil, err
	}
	p := s.Project(id)
	if p == nil {
		return nil, fmt.Errorf("%s is not a project yet\nRegister it with `whatagain projects add`", id)
	}
	return p, nil
}

// currentProject returns the registered project for the working directory,
// or nil when there is none. Used where an unknown project is a fallback
// rather than an error.
func currentProject(s *store.Store) *store.Project {
	id, err := repo.Current()
	if err != nil {
		return nil
	}
	return s.Project(id)
}
