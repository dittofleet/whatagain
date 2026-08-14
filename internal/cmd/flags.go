package cmd

import (
	"fmt"
	"strings"
)

// flags declares what a command accepts. Keys are flag names without
// dashes. Registering several keys against the same pointer gives a flag
// its aliases (e.g. "p" and "project").
type flags struct {
	bools  map[string]*bool
	values map[string]*string
}

// projectFlag and yesFlag build the two flag sets shared across commands,
// so an alias is spelled once rather than at every call site.
func projectFlag(target *string) map[string]*string {
	return map[string]*string{"p": target, "project": target}
}

func descriptionFlag(target *string) map[string]*string {
	return map[string]*string{"d": target, "desc": target, "description": target}
}

func yesFlag(target *bool) map[string]*bool {
	return map[string]*bool{"y": target, "yes": target}
}

// parse pulls the declared flags out of args and returns the positionals.
// Flags may appear anywhere, and a bare "--" ends flag parsing so item
// text starting with a dash can still be written literally.
func (f flags) parse(args []string, usage string) ([]string, error) {
	positional := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if len(arg) < 2 || !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}

		name, inline, hasInline := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		switch target, isBool := f.bools[name]; {
		case isBool:
			if hasInline {
				return nil, fmt.Errorf("flag takes no value: %s\n%s", arg, usage)
			}
			*target = true
		case f.values[name] != nil:
			value := inline
			if !hasInline && i+1 < len(args) {
				i++
				value = args[i]
			}
			// An empty value is rejected rather than ignored: a script
			// passing an unset variable should not quietly act on
			// whatever the default target happens to be.
			if strings.TrimSpace(value) == "" {
				return nil, fmt.Errorf("flag needs a value: %s\n%s", arg, usage)
			}
			*f.values[name] = value
		default:
			return nil, fmt.Errorf("unknown flag: %s\n%s", arg, usage)
		}
	}
	return positional, nil
}
