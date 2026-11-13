# E12 · Syncing

**You will learn:** sync by hand, on open and on a timer.

`space s` or `:sync` runs a full sync: it commits whatever is pending, fetches from the remote, merges and pushes. While it runs, the bar says so; when it ends, `ok` or the reason it stopped. It is the button you will press when you want to secure your work before closing the laptop.

There are two more automatic syncs. One when you open a vault on the list: if you go into your notes, Misoku syncs once without asking. And one on a timer, every five minutes by default, while the app is open. That interval is changed per vault with `interval`, with a minimum of one minute; `off` turns it off and keeps the other two.

All three do the same thing inside. The difference is who triggers them: you, opening the vault, or the clock.

## Your turn

1. Press `space s` (or `:sync`) and wait for the end.
2. Open a vault on the list again and watch the sync on open.
3. Wait for the timed sync, or lower `interval` so you do not wait five minutes.



Next: [[E13 What sync does]]
