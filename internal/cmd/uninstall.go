package cmd

import (
	"fmt"
	"os"

	clikit "github.com/dittofleet/go-cli-kit"
	"github.com/dittofleet/go-cli-kit/uninstall"
	"github.com/dittofleet/go-cli-kit/xdg"
	"github.com/dittofleet/whatagain/internal/store"
)

const uninstallUsage = "usage: whatagain uninstall [--yes]"

// Uninstall removes the whatagain binary, config directory, and data
// directory.
func Uninstall(args []string, a clikit.App) error {
	var yes bool
	args, err := flags{bools: yesFlag(&yes)}.parse(args, uninstallUsage)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %v\n%s", args, uninstallUsage)
	}
	return uninstall.Run(a, yes, uninstall.Plan{
		Items: []uninstall.Item{
			{Label: "Cache", Path: xdg.DataDir(a.Name), Remove: os.RemoveAll},
			{Label: "Config", Path: xdg.ConfigDir(a.Name), Note: describeStore(), Remove: os.RemoveAll},
		},
		Notice: "If the config directory is synced between machines, deleting it here removes your items everywhere.",
	})
}

// describeStore summarizes what is about to be deleted. An unreadable
// store is not worth failing the uninstall over.
func describeStore() string {
	s, err := store.Load()
	if err != nil {
		return "your projects and items"
	}
	// Lists, not Projects: items on the global list are as gone as any
	// other once the store is deleted.
	return plural(len(s.Projects), "project") + ", " + plural(itemCount(s.Lists()), "item")
}
