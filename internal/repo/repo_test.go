package repo

import "testing"

func TestParseGitHubURL(t *testing.T) {
	ok := map[string]string{
		"git@github.com:dittofleet/whatagain.git":          "dittofleet/whatagain",
		"git@github.com:dittofleet/whatagain":              "dittofleet/whatagain",
		"https://github.com/dittofleet/whatagain.git":      "dittofleet/whatagain",
		"https://github.com/dittofleet/whatagain":          "dittofleet/whatagain",
		"https://github.com/dittofleet/whatagain/":         "dittofleet/whatagain",
		"ssh://git@github.com/dittofleet/whatagain.git":    "dittofleet/whatagain",
		"git://github.com/dittofleet/whatagain.git":        "dittofleet/whatagain",
		"https://user@github.com/dittofleet/shigoto.git":   "dittofleet/shigoto",
		"https://github.com/dittofleet/dots.with.dots.git": "dittofleet/dots.with.dots",
		"https://GitHub.com/dittofleet/whatagain.git":      "dittofleet/whatagain",
	}
	for url, want := range ok {
		got, err := ParseGitHubURL(url)
		if err != nil {
			t.Errorf("ParseGitHubURL(%q) errored: %v", url, err)
			continue
		}
		if got != want {
			t.Errorf("ParseGitHubURL(%q) = %q, want %q", url, got, want)
		}
	}

	bad := []string{
		"git@gitlab.com:dittofleet/whatagain.git",
		"https://github.com/dittofleet",
		"https://github.com/dittofleet/whatagain/extra",
		"https://github.com/",
		"",
		// Hosts that merely contain the string "github.com".
		"https://notgithub.com/evil/repo.git",
		"https://github.com.evil.com/evil/repo.git",
		"git@github.example.com:dittofleet/whatagain.git",
	}
	for _, url := range bad {
		if got, err := ParseGitHubURL(url); err == nil {
			t.Errorf("ParseGitHubURL(%q) = %q, want an error", url, got)
		}
	}
}
