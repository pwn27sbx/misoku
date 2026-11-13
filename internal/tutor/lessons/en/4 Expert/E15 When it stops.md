# E15 · When it stops

**You will learn:** the three things that stop sync and how to clear them.

When sync cannot go on, Misoku stops and explains it in plain words, without dumping a raw git error. The dialog says what happened and offers "Check again" to retry once you have fixed it; it does not keep retrying on its own, and it does not lose what it had already done.

The most common stop is an orphaned git lock: an `index.lock` left behind by an interrupted process. While it exists, git refuses to work. The dialog offers "Remove the lock and sync", which deletes that file and carries on where it was.

The other one is an expired GitHub credential on an HTTPS remote: instead of failing silently, the app offers "Sign in", which runs `gh auth login`. You accept, sign in in the terminal and the next sync works.

None of the three clears itself: the app asks before touching anything.

## Your turn

1. Press `space s` with a simulated problem, read the dialog and use "Check again".
2. Leave a git lock behind and accept "Remove the lock and sync": the file disappears and sync carries on.
3. With an expired GitHub remote, choose "Sign in" to run `gh auth login`.

Write down below which stop you got and how you cleared it.



Next: [[E16 Notes that arrive]]
