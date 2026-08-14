# whatagain

**NOTE** this project was created for personal use. I am unable to guarantee the quality or polish that one may expect from a properly maintained project.

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
