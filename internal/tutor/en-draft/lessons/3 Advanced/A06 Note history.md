# A06 · Note history

**You will learn:** to see what changed in a note, compare versions and bring old text back into the editor without losing what you have now.

`:history` asks the vault's git repository for the commits that touched this note, renames included, and lists them with their date beside them. It is not a history of Misoku's own: it is git's, so the vault has to be a repository with commits, like the one sync creates.

As you move through the list with `j` and `k`, or with the wheel, the panel beside it shows that version. It is a read, not a change: the note stays as it is.

`Enter` on a version puts its text into the editor as one undoable change. Nothing is written until you save, and if you change your mind, `u` reverts it in one go. In a note with no commits, or in a vault without git, there is no dialog: the bar says why and nothing else happens. The tutorial vault has no git, so that notice is what you will see; the restore steps are practised in a vault with sync.

## Your turn

1. In a vault with commits, open a note that has changed and run `:history`.
2. Move with `j` and `k` (or with the wheel) and look at the version that appears beside it.
3. Add `budget: 900` to the front matter, save, change it to `budget: 950`, save and, once there are commits, press `Enter` on the old version and undo with `u`.
4. Run `:history` in a note with no commits and read the notice in the bar.



Next: [[A07 Mentions and backlinks]]
