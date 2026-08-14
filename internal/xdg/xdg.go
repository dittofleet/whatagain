// Package xdg resolves per-app XDG Base Directory paths.
//
// If $XDG_*_HOME is unset and os.UserHomeDir fails, home falls back to
// "/" and the returned path becomes "/.config/<app>" or
// "/.local/share/<app>". Downstream file operations will surface a clear
// permission error pointing at the bad path, which is more useful in a
// CLI context than a generic "home not found" error, and far more useful
// than a relative path quietly rooting a second store in the working
// directory.
package xdg

import (
	"os"
	"path/filepath"
)

// App is the directory name every part of the tool stores things under,
// kept here so the config and data dirs cannot drift apart.
const App = "whatagain"

// ConfigDir returns $XDG_CONFIG_HOME/<app> or ~/.config/<app>.
func ConfigDir(app string) string {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, app)
	}
	return filepath.Join(home(), ".config", app)
}

// DataDir returns $XDG_DATA_HOME/<app> or ~/.local/share/<app>.
func DataDir(app string) string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, app)
	}
	return filepath.Join(home(), ".local", "share", app)
}

// home resolves the home directory, falling back to the root so callers
// always build an absolute path. Joining onto "" would yield a relative
// one, which would put a separate store under every working directory
// and aim uninstall's deletions at the working directory too.
func home() string {
	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		return "/"
	}
	return dir
}
