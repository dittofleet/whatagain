package repo

import "testing"

func TestParseGitHubURL(t *testing.T) {
	ok := map[string]string{
		"git@github.com:sylophi/whatagain.git":          "sylophi/whatagain",
		"git@github.com:sylophi/whatagain":              "sylophi/whatagain",
		"https://github.com/sylophi/whatagain.git":      "sylophi/whatagain",
		"https://github.com/sylophi/whatagain":          "sylophi/whatagain",
		"https://github.com/sylophi/whatagain/":         "sylophi/whatagain",
		"ssh://git@github.com/sylophi/whatagain.git":    "sylophi/whatagain",
		"git://github.com/sylophi/whatagain.git":        "sylophi/whatagain",
		"https://user@github.com/sylophi/shigoto.git":   "sylophi/shigoto",
		"https://github.com/sylophi/dots.with.dots.git": "sylophi/dots.with.dots",
		"https://GitHub.com/sylophi/whatagain.git":      "sylophi/whatagain",
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
		"git@gitlab.com:sylophi/whatagain.git",
		"https://github.com/sylophi",
		"https://github.com/sylophi/whatagain/extra",
		"https://github.com/",
		"",
		// Hosts that merely contain the string "github.com".
		"https://notgithub.com/evil/repo.git",
		"https://github.com.evil.com/evil/repo.git",
		"git@github.example.com:sylophi/whatagain.git",
	}
	for _, url := range bad {
		if got, err := ParseGitHubURL(url); err == nil {
			t.Errorf("ParseGitHubURL(%q) = %q, want an error", url, got)
		}
	}
}
