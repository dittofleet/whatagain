---
name: whatagain
description: The user's per-repo todo list. Use when they ask to note something for later, or what is on the list.
---

# whatagain: the user's todo list, per repo

A scratch list of things the user thought of and wants to get to later, kept per GitHub repo. It is their list, not a place to track the work you are doing right now.

```sh
whatagain help
```

## Notes

- Only add what the user asks you to note. Do not file your own observations, follow-ups, or leftover TODOs.
- An item can carry a description (`add -d "<text>"`, or `desc <id> "<text>"` later). It is optional: use it when the user gave detail worth keeping, not to pad a note they meant to be short.
- An item can carry tags (`add -t <tag>`, or `tag <id> <tag>...` later), and `ls -t <tag>` shows only the items carrying it. Tags are the user's words, so use the ones they said rather than inventing a scheme of your own.
- A note that is about no repo goes on the global list with `add -g`, not into whichever repo the session happens to be in. `ls -g` shows that list, and the by-id commands need no flag there.
- Removing is permanent. There is no done state, no history, and no undo, so remove an item only when the user says it is finished or asks you to drop it.
- Remove by id, taken from `whatagain ls`, rather than by text. It is exact, and it works from any directory.
- `add` in an unregistered repo is an error. Whether a repo belongs on the list is the user's call, so ask before running `whatagain projects add`.
- A repo registered in [terrier](https://github.com/dittofleet/terrier) becomes a project here on its own, so `add` works there with nothing to register. `terrier add` is as much the user's call as `whatagain projects add`, so ask before either.
- `command not found`: tell the user, and do not try to install it yourself.
