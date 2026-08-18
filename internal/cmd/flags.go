package cmd

import (
	"fmt"
	"strings"
)

// flags declares what a command accepts. Keys are flag names without
// dashes. Registering several keys against the same pointer gives a flag
// its aliases (e.g. "p" and "project"). A flag in lists collects every
// value it is given instead of keeping only the last.
type flags struct {
	bools  map[string]*bool
	values map[string]*string
	lists  map[string]*[]string
}

// projectFlag and yesFlag build the two flag sets shared across commands,
// so an alias is spelled once rather than at every call site.
func projectFlag(target *string) map[string]*string {
	return map[string]*string{"p": target, "project": target}
}

func descriptionFlag(target *string) map[string]*string {
	return map[string]*string{"d": target, "desc": target, "description": target}
}

// tagFlag collects rather than overwrites, so -t ci -t flaky reads the way
// it looks. Each value can also be a comma-separated list. Splitting them
// is parseTags's job, since the same text arrives as positionals too.
func tagFlag(target *[]string) map[string]*[]string {
	return map[string]*[]string{"t": target, "tag": target, "tags": target}
}

func yesFlag(target *bool) map[string]*bool {
	return map[string]*bool{"y": target, "yes": target}
}

// declares reports whether arg is one of this command's flags, or the "--"
// that ends them. Text that merely starts with a dash is not: a
// description can begin with one.
func (f flags) declares(arg string) bool {
	if arg == "--" {
		return true
	}
	if len(arg) < 2 || !strings.HasPrefix(arg, "-") {
		return false
	}
	name, _, _ := strings.Cut(strings.TrimLeft(arg, "-"), "=")
	return f.bools[name] != nil || f.values[name] != nil || f.lists[name] != nil
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
		if target, isBool := f.bools[name]; isBool {
			if hasInline {
				return nil, fmt.Errorf("flag takes no value: %s\n%s", arg, usage)
			}
			*target = true
			continue
		}

		single, list := f.values[name], f.lists[name]
		if single == nil && list == nil {
			return nil, fmt.Errorf("unknown flag: %s\n%s", arg, usage)
		}
		value := inline
		// Another flag is never the value, so an unset variable in
		// `ls -t $TAG --json` is an error rather than a filter on the word
		// "--json", which would quietly match nothing.
		if !hasInline && i+1 < len(args) && !f.declares(args[i+1]) {
			i++
			value = args[i]
		}
		// An empty value is rejected rather than ignored: a script passing
		// an unset variable should not quietly act on whatever the default
		// target happens to be.
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("flag needs a value: %s\n%s", arg, usage)
		}
		if list != nil {
			*list = append(*list, value)
			continue
		}
		*single = value
	}
	return positional, nil
}
