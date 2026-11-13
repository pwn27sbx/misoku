# E14 · The sync status

**You will learn:** the screen that tells you everything sync is doing.

`:sync status` opens a read-only window with the full status, one field per line: State (the same words as the status bar), Remote (with the credential hidden), Branch, Timed sync (how often it syncs), Pending (commits left to push), Last synced (once this window has synced), Last error, and the Conflict copies you need to resolve.

At the bottom, two lists explain the odd cases. "Files that do not sync as they are" are the files git cannot upload as they are: over 50 MB, ignored by `.gitignore`, with names Windows does not accept, or inside another nested repository. And the list of notes the last sync brought, such as "2 new notes from another computer", reminds you of what arrived, in case it came while you were looking at something else.

It is the first stop when something does not add up: the reason is almost always written there before you look for it anywhere else.

## Your turn

1. Run `:sync status` and read State, Remote, Branch, Timed sync, Pending, Last synced, Last error and Conflict copies.
2. On that same screen, look at the list of files left out: large, ignored or with illegal names.



Next: [[E15 When it stops]]
