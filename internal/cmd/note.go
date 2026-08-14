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

// normalizeDescription tidies a description without flattening it. Unlike a
// note, which stays one line so listings and text matching are predictable,
// a description is free to keep the line breaks that make longer detail
// readable. Only trailing whitespace and surrounding blank lines go.
func normalizeDescription(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
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
