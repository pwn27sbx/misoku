# A04 · Note status

**You will learn:** to give each note a status and to recognise it at a glance, without opening it.

The status is one more property: `status:` with one of four values. `active` is a note in progress, `on hold` is parked, `completed` is finished and `dropped` was abandoned. They are written in the front matter and saved like any other data.

As soon as the status is saved, the explorer and the home screen replace the note's icon with a glyph: a dot for active, a half circle for on hold, a check mark for completed and a cross for dropped. That way the tree reads like a board.

A value that is not one of those four is not an error: the note is treated as if it had no status and gets its normal icon back. It lets you experiment without breaking anything.

## Your turn

1. Add `status: active` to the front matter and save the note.
2. Change the status to `on hold`, to `completed` and to `dropped`, saving each time, and look at the glyph in the explorer and on the home screen.
3. Write `status: whatever`, save and check that the note is left without a glyph.



Next: [[A05 Advanced filters]]
