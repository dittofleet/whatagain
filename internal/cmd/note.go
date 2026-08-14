package cmd

import (
	"fmt"
	"strings"
)

// normalizeNote collapses whitespace so every item stays a single line,
// which is what makes `ls` output and text matching predictable.
func normalizeNote(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// quotedSuggestion renders the command the user probably meant, for when
// the shell has already split their note into separate arguments. It gives
// up on a note containing a double quote rather than print a broken
// suggestion.
func quotedSuggestion(command string, args []string) string {
	text := normalizeNote(strings.Join(args, " "))
	if text == "" || strings.Contains(text, `"`) {
		return ""
	}
	// Text arriving after a "--" can still start with a dash, and the
	// suggestion has to keep the terminator to run.
	terminator := ""
	if strings.HasPrefix(text, "-") {
		terminator = "-- "
	}
	return fmt.Sprintf("\n  whatagain %s %s%q", command, terminator, text)
}
