package terrier

import "testing"

func TestCompatible(t *testing.T) {
	// Everything up to and including what this was built against, plus a
	// build from source, which is somebody running their own on purpose.
	for _, v := range []string{"v0.1.0", "v0.1.1", "0.1.9", "v0.0.4", devVersion} {
		if err := compatible(v); err != nil {
			t.Errorf("compatible(%q) errored: %v", v, err)
		}
	}

	// A newer minor means something a tool could be relying on has moved.
	for _, v := range []string{"v0.2.0", "v1.0.0", "v0.10.0"} {
		if err := compatible(v); err == nil {
			t.Errorf("compatible(%q) = nil error, want a terrier this build does not understand", v)
		}
	}

	for _, v := range []string{"", "v1", "version 2", "v0.x.1"} {
		if err := compatible(v); err == nil {
			t.Errorf("compatible(%q) = nil error, want an unreadable version", v)
		}
	}
}

func TestParseVersionComparesNumbersNotText(t *testing.T) {
	// The one comparison that text gets wrong.
	_, minor, err := parseVersion("v0.10.0")
	if err != nil {
		t.Fatalf("parseVersion errored: %v", err)
	}
	if minor != 10 {
		t.Errorf("minor = %d, want 10", minor)
	}
}
