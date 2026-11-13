# A20 · Editing folders in the editor

**You will learn:** to create, rename, move and delete files as if they were lines of text, and to step out to mini.files when you need to.

In the explorer, `m` opens the selected folder as a file of lines: one per entry. Writing a line creates, editing a name renames, deleting the line deletes, and `dd` with `p` in another folder moves. Nothing touches the disk while you type.

When you confirm with `=`, Misoku shows the plan: what is created, what is renamed and what goes to the Trash, with the links that would need updating. Only after you accept is it applied. Deleting sends to the system Trash, it does not destroy.

From the tree, `i`, `a` and `o` open the editor in the selected note's folder with that key already received, so the gesture is not lost. And `M` hands the screen to mini.files, the Neovim file manager: when you close it, Misoku rereads the folder and whatever you did shows up.

## Your turn

1. In the explorer create the folder `Notes` with `boots.md` and `old.md` inside, select the folder and press `m`.
2. In the line editor create `luggage.md`, rename `boots.md` to `shoes.md`, delete `old.md`, press `=` and confirm.
3. In the tree, press `i`, `a` and `o` on a note.
4. Press `M`, create `practice-mini.md` with mini.files, close it and see that it appears in Misoku.



Next: [[A21 Advanced Challenge]]
