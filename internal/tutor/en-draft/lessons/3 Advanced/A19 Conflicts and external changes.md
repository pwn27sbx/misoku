# A19 · Conflicts and external changes

**You will learn:** what happens when the file changes from outside while you have it open, and how Misoku avoids losing text.

If you save a note whose file changed on disk, both versions count. Misoku does not choose: it writes the text from disk into a conflict copy beside it, with the date in the name, and leaves yours in the note. The bar says so with `Both kept:` and the copy's name, so you know where the other one went.

The same holds for a note you delete or rename from outside. The tab keeps its text, warns you once and, if you try to save, asks before creating the file again. Nothing is recreated silently.

Sync uses the same idea between two copies of the vault: when both touch the same lines, the conflict copy keeps one version and the note the other, and you decide which one stays. Two texts, two files, nothing lost.

## Your turn

1. With this note open, change it from outside (another editor or an `echo`) and save here: look at `Both kept:` in the bar and at the conflict copy next to the note.
2. Delete or rename a note from outside, go back to Misoku and check that it warns once and that saving asks before recreating it.
3. With sync set up on two copies of the vault, cause a conflict on the same line and look at the copy `plan (conflict …).md`.
4. Check that your text is in the note and the other one in the copy.



Next: [[A20 Editing folders in the editor]]
