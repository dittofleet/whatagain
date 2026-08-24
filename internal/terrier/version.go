package terrier

import (
	"fmt"
	"strconv"
	"strings"
)

// The terrier version whatagain is written against. A minor bump means
// something a tool could be relying on has changed, so only those two
// components are recorded and compared: a patch bump never changes
// anything a tool can see.
const (
	wantMajor = 0
	wantMinor = 1
)

// devVersion is what a terrier built from source reports.
const devVersion = "dev"

func checkVersion() error {
	out, err := run("version")
	if err != nil {
		return err
	}
	return compatible(string(out))
}

// compatible reports whether a terrier reporting this version can be
// relied on.
//
// Only a newer minor is refused. Terrier only ever adds to what it
// prints, so an older one still answers the two questions asked of it,
// and there is nothing to gain by insisting on the latest.
//
// A terrier built from source reports "dev", which has nothing to compare
// against. That is somebody running their own build on purpose, and
// second-guessing them would only take their projects away.
func compatible(version string) error {
	v := strings.TrimSpace(version)
	if v == devVersion {
		return nil
	}
	major, minor, err := parseVersion(v)
	if err != nil {
		return err
	}
	if major > wantMajor || (major == wantMajor && minor > wantMinor) {
		return fmt.Errorf("terrier %s is newer than this whatagain understands (built against %d.%d), so its projects are being left out\nUpdate with `whatagain update`", v, wantMajor, wantMinor)
	}
	return nil
}

// parseVersion splits a "v0.1.0" tag into its major and minor components.
// They come back as numbers because compared as text 0.10 would read as
// older than 0.9.
func parseVersion(v string) (major, minor int, err error) {
	fail := fmt.Errorf("could not read the terrier version: %q is not one", v)
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) < 2 {
		return 0, 0, fail
	}
	if major, err = strconv.Atoi(parts[0]); err != nil {
		return 0, 0, fail
	}
	if minor, err = strconv.Atoi(parts[1]); err != nil {
		return 0, 0, fail
	}
	return major, minor, nil
}
