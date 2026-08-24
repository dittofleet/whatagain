package cmd

import (
	"github.com/dittofleet/whatagain/internal/store"
	"github.com/dittofleet/whatagain/internal/terrier"
)

// Every command reads the store through openStore or writes it through
// updateStore, so terrier is checked wherever the list is about to be
// used and a repo registered there is a project here from that moment on.
//
// Adopting rather than merging on the way past is what makes the list the
// same on every machine. Terrier registers paths, so its registry is
// local to the machine it was written on, while the store is the thing
// that syncs. A repo registered on one machine has to land in the store
// to be a project on the others.
//
// Nothing goes the other way. Unregistering a repo in terrier says where
// it is no longer worth keeping a path to, which is a smaller thing than
// saying the notes hanging off it should be deleted on every machine you
// own.

// openStore reads the store, with any repo terrier has registered and it
// does not have yet added to it.
func openStore() (*store.Store, error) {
	slugs := terrier.Slugs()
	s, err := store.Load()
	if err != nil {
		return nil, err
	}
	// Adopting is a write, and almost every invocation has nothing to
	// adopt. Trying it on what was just read answers that for free: when
	// nothing was new, this stays the single read it has always been, with
	// no lock taken and no file rewritten for a dotfile sync to carry.
	if len(adopt(s, slugs)) == 0 {
		return s, nil
	}
	// Something was new, so this read has a write to do after all. What
	// was adopted above is thrown away and done again under the lock, on a
	// fresh read, so a whatagain writing in another worktree at the same
	// moment is not rolled back to whatever this one happened to load. A
	// write with nothing of its own to do is what updateStore already is.
	var synced *store.Store
	if err := updateStore(func(s *store.Store) error {
		synced = s
		return nil
	}); err != nil {
		// Adopting is something this read does on the way past, not what
		// it was asked for, so failing to write it is not worth failing
		// the command over: a store on a read-only disk, or one another
		// whatagain is holding the lock on, would take `ls` down with it.
		// The copy in hand already has the projects, and the next command
		// that can write is the one that keeps them.
		return s, nil
	}
	return synced, nil
}

// updateStore is store.Update with the same repos folded in first, so one
// lock and one write cover both adopting them and whatever the command
// went on to do. A command that fails saves neither, and the next one
// adopts them again.
func updateStore(fn func(*store.Store) error) error {
	slugs := terrier.Slugs()
	return store.Update(func(s *store.Store) error {
		adopt(s, slugs)
		return fn(s)
	})
}

// adopt gives the store a project for every repo terrier has registered
// that it does not already hold, and reports the ones it added.
func adopt(s *store.Store, slugs []string) []string {
	var added []string
	for _, slug := range slugs {
		// AddProject refuses a project the store already has, and anything
		// that is not an owner/name slug. Both are exactly what adopting
		// should pass over, so its error is the check rather than a
		// failure: whatagain does not stop working because terrier has
		// something in it that cannot be a project here.
		if _, err := s.AddProject(slug); err == nil {
			added = append(added, slug)
		}
	}
	return added
}
