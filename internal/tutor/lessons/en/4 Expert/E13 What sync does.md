# E13 · What sync does

**You will learn:** the two moments when git works without you asking.

Saving a note does not write the commit on the spot. A few seconds after your last save, Misoku makes a local commit with the changes: no network, just `git commit`. That way the history moves forward while you write, and a later sync only has to push what is already committed.

When you close Misoku the opposite happens: the app pushes what is pending for up to ten seconds after the screen disappears. If the remote answers, your notes go out on their own; if not, they stay in local commits waiting for the next sync, and the bar will tell you when you open it again.

The practical consequence: closing the app never loses work, but it does not always publish it. If you want to be sure, open it, look at the status and let it finish.

## Your turn

1. Save a note and wait a few seconds: the git history has a local commit with that change.
2. Close Misoku and see that it pushes for up to ten seconds after the screen closes.

Write down below what you see in the bar; that way the tutorial knows you went through the step.



Next: [[E14 The sync status]]
