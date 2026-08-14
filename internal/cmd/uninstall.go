package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/dittofleet/whatagain/internal/store"
	"github.com/dittofleet/whatagain/internal/xdg"
	"golang.org/x/term"
)

const uninstallUsage = "usage: whatagain uninstall [--yes]"

// Uninstall removes the whatagain binary, config directory, and data
// directory. Order is data → config → binary so a failure leaves a tool
// to retry with.
func Uninstall(args []string, version string) error {
	var yes bool
	args, err := flags{bools: yesFlag(&yes)}.parse(args, uninstallUsage)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %v\n%s", args, uninstallUsage)
	}

	if version == "dev" {
		return errors.New("cannot uninstall a dev build")
	}

	binaryPath, err := resolveExecutable()
	if err != nil {
		return fmt.Errorf("cannot determine binary path: %w", err)
	}

	configDir := xdg.ConfigDir(xdg.App)
	dataDir := xdg.DataDir(xdg.App)

	fmt.Println("This will remove:")
	fmt.Printf("  - Binary:  %s\n", binaryPath)
	fmt.Printf("  - Config:  %s  (%s)\n", configDir, describeStore())
	fmt.Printf("  - Cache:   %s\n", dataDir)
	fmt.Println()
	fmt.Println("If the config directory is synced between machines, deleting it here removes your items everywhere.")
	fmt.Println()

	if !yes {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return errors.New("refusing to uninstall non-interactively without --yes")
		}
		fmt.Print("Proceed? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		answer := strings.ToLower(strings.TrimSpace(line))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	steps := []struct {
		label string
		path  string
		fn    func(string) error
	}{
		{"cache directory", dataDir, os.RemoveAll},
		{"config directory", configDir, os.RemoveAll},
		{"binary", binaryPath, os.Remove},
	}
	var removed []string
	for _, s := range steps {
		err := s.fn(s.path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			if len(removed) > 0 {
				fmt.Fprintf(os.Stderr, "Removed before failure: %s\n", strings.Join(removed, ", "))
			}
			return fmt.Errorf("failed to remove %s (%s): %w", s.label, s.path, err)
		}
		removed = append(removed, s.label)
	}

	fmt.Println("Uninstalled whatagain.")
	return nil
}

// describeStore summarizes what is about to be deleted. An unreadable
// store is not worth failing the uninstall over.
func describeStore() string {
	s, err := store.Load()
	if err != nil {
		return "your projects and items"
	}
	return plural(len(s.Projects), "project") + ", " + plural(itemCount(s.Projects), "item")
}
