# whatagain

A todo list for coding agents, scoped to repos.

Note something down the moment you think of it, from whatever repo or worktree you happen to be in. `whatagain` works out which project you mean from the `origin` remote, so there is nothing to select and nothing to configure.

```sh
$ whatagain add "fix the flaky land test"
Added 2437 to dittofleet/shigoto-no-mori: fix the flaky land test

$ whatagain ls
dittofleet/shigoto-no-mori
  2437  fix the flaky land test
  cae3  ship the windows build

$ whatagain rm 2437
Removed 2437 from dittofleet/shigoto-no-mori: fix the flaky land test
```

When one line does not say enough, hang a description off the item. It is optional everywhere, so items without one stay the single line they always were:

```sh
$ whatagain add "fix the flaky land test" -d "only fails in CI, suspect the temp dir"
$ whatagain desc cae3 "needs a signing cert first"

$ whatagain ls
dittofleet/shigoto-no-mori
  2437  fix the flaky land test
        only fails in CI, suspect the temp dir
  cae3  ship the windows build
        needs a signing cert first
```

Drop a description again with `whatagain desc cae3 --clear`.

Tags are words you hang on an item to find it again. Nothing registers them, and there is no list of them to keep tidy: a tag exists as long as some item carries it.

```sh
$ whatagain add "sign the installer" -t windows,release
$ whatagain tag cae3 windows

$ whatagain ls -t windows
dittofleet/shigoto-no-mori
  cae3  ship the windows build  #windows
  7d10  sign the installer  #windows #release
```

Filter on several tags and you get the items carrying all of them, so each one you add narrows the list. Take a word back off with `whatagain untag cae3 windows`, or drop them all with `whatagain tag cae3 --clear`.

Run `whatagain help` for the rest of the commands.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/dittofleet/whatagain/main/install.sh | sh
```

Installs the latest release to `~/.local/bin/whatagain` (override with `WHATAGAIN_INSTALL_DIR`). Supported platforms: macOS (arm64, x64), Linux (arm64, x64).

## The store

Everything lives in one JSON file at `~/.config/whatagain/todo.json`, created on the first write. Sync it with [lichen](https://github.com/dittofleet/lichen), or anything else that syncs dotfiles, and the list follows you between machines:

```sh
lichen sync ~/.config/whatagain/todo.json
```

## Agent skill

`skills/whatagain/SKILL.md` tells a coding agent what the list is for and when to reach for it, so "note that down for later" lands in the right project without you spelling out the command.
