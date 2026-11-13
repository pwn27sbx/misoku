# E16 · Notes that arrive

**You will learn:** to find out what sync brings without opening anything.

The status bar shows sync in four short states: `syncing…` while it works, `synced 5m ago` with the time of the last success, `N conflicts` if there are unresolved conflict copies, and `offline · N pending` when there is no network and commits are waiting. You read them at a glance and they do not interrupt what you are writing.

When a sync brings new notes, the bar names up to three and `:messages` keeps up to ten, with their time. It is the "this arrived while you were not looking" notice: if you were in another note, there it is.

And only one window syncs each vault. If you open a second instance on the same notes, the new one says so in the bar with `other window syncs` and does not compete for the repository: two processes pushing at once would be the easiest way to create conflicts.

## Your turn

1. Look at the bar: `syncing…`, `synced 5m ago`, `N conflicts`, `offline · N pending`.
2. Make a note arrive and read the notice: three names in the bar, ten in `:messages`.
3. Open a second Misoku window on the same vault and see what the bar says.



Next: [[E17 Setting up sync]]
