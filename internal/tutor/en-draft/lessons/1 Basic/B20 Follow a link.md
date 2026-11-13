# B20 · Follow a link

**You will learn:** the four ways to jump to another note, which ones open a new tab, how to come back with `ctrl+o`, and what happens if the note doesn't exist.

With the cursor on a link and in NORMAL, `gd` opens it in the same tab, as in Neovim, and `gx` does the same; if your tab has unsaved changes, it is kept and the note opens next to it. `Enter`, as in LazyVim, and a click from the preview open it in a new tab. If the note is already open, you just switch to its tab. If the target doesn't exist, `gd` and `gx` open a draft with that name for you to write, while `Enter` only tells you there is no note. That difference matters: `Enter` never creates a note.

`ctrl+o` takes the tab back to the previous note, at the line where the cursor was, and each press goes back one more. It doesn't work in INSERT.

## Your turn

1. With the cursor on the link in the practice area, press `gd`.
2. Come back with `ctrl+o` and press `gx` on the same link.
3. Press `Enter` on the link.
4. Click the link from the preview.
5. Type `[[Beach list]]` on a new line and press `gd`: the note doesn't exist and a draft opens.

## Your practice area

Read: [[After the tutorial]] gathers the terminal commands and what the routes don't practise.

To come back to this note after jumping: if it was in the same tab, press `ctrl+o`; if a new tab opened, use `H` or the lesson's tab in the bar.

Next: [[B21 Read without editing]]
