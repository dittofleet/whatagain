package main

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/dittofleet/whatagain/internal/cmd"
	"github.com/dittofleet/whatagain/internal/update"
)

var errUnknownCommand = errors.New("unknown command")

var version = "dev"

const usage = `Usage: whatagain <command>

Commands:
  add "<text>"             Note something down for the current project
  desc <id> "<text>"       Add detail to an item, or --clear what it has
  tag <id> <tag>...        Hang words on an item, or --clear the ones it has
  untag <id> <tag>...      Take words back off an item
  rm <id>... | "<text>"    Remove items by id, or by what they say
  ls                       List the current project's items
  projects                 List projects and their item counts
  projects add [<repo>]    Register a project, defaulting to the current repo
  projects rm <repo>...    Unregister a project and drop its items
  update                   Download and install the latest version
  uninstall [--yes]        Remove the binary, config, and cache
  version                  Print the installed version
  help                     Print this help message

Flags:
  -p, --project <repo>     Act on a project other than the current repo
  -d, --desc <text>        (add) Optional detail to go with the note
  -t, --tag <tag>          (add) Tag the note, (ls) show only what carries it
      --clear              (desc, tag) Drop the item's detail or its tags
      --all                (ls) Show every project
      --json               (ls, projects) Machine-readable output
      --                   Everything after this is text, not flags

Notes are one quoted argument, so the shell hands them over intact. A note is
one line. A description can run to several, and is optional everywhere.

A tag is a word hung on an item, and nothing registers it: it exists as long as
some item carries it. Write tags as bare words, comma-separated or one -t each.
Filtering on several shows the items carrying all of them.

A project is a GitHub repo, e.g. dittofleet/whatagain. The current one comes
from the ` + "`origin`" + ` remote of the git repository or worktree you are in, so
worktrees resolve to the same project as the primary checkout. A bare ` + "`ls`" + `
outside a registered repo lists everything.
`

func printUsage() {
	fmt.Print(usage)
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		printUsage()
		os.Exit(0)
	}

	if err := dispatch(args); err != nil {
		if errors.Is(err, errUnknownCommand) {
			// Naming it catches the common slip of putting a flag before
			// the command, where a bare usage dump explains nothing.
			fmt.Fprintf(os.Stderr, "Error: unknown command: %s\n\n", args[0])
			printUsage()
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}

	// `update` has just talked to the release API, and `uninstall` has
	// deleted the cache directory this would recreate.
	if !slices.Contains([]string{"update", "uninstall"}, args[0]) {
		update.MaybeCheck(version)
	}
}

func dispatch(args []string) error {
	switch args[0] {
	case "add":
		return cmd.Add(args[1:])
	case "desc", "describe":
		return cmd.Describe(args[1:])
	case "tag":
		return cmd.Tag(args[1:])
	case "untag":
		return cmd.Untag(args[1:])
	case "rm", "remove":
		return cmd.Remove(args[1:])
	case "ls", "list":
		return cmd.List(args[1:])
	case "projects", "project":
		return cmd.Projects(args[1:])
	case "update":
		return cmd.SelfUpdate(version)
	case "uninstall":
		return cmd.Uninstall(args[1:], version)
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return errUnknownCommand
	}
}
