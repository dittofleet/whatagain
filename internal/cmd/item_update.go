package cmd

import (
	"fmt"

	"github.com/dittofleet/whatagain/internal/store"
)

// updateItem is the frame every command that addresses an item by id
// shares: open the store, find the item, hand it to change, and report the
// project it turned out to live in. Ids are unique store-wide, so none of
// those commands needs a project to work from.
func updateItem(id string, change func(p *store.Project, i int) (store.Item, error)) (store.Item, string, error) {
	var item store.Item
	var target string
	err := store.Update(func(s *store.Store) error {
		p, i := s.FindItemByID(id)
		if p == nil {
			return fmt.Errorf("no item with id: %s", id)
		}
		updated, err := change(p, i)
		if err != nil {
			return err
		}
		item, target = updated, p.ID
		return nil
	})
	return item, target, err
}
