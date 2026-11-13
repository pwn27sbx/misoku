# B05 · Undo and redo

**You will learn:** the safety net: `u` undoes and `ctrl+r` redoes.

Every insertion or deletion is a change, and `u` undoes the last one; holding it down walks the history backwards. `ctrl+r` does the opposite. It is the reason you don't need to be afraid in vim: almost anything can be fixed with one key press.

## Your turn

1. Put the cursor on the line `delete me` and press `dd`: the line disappears. Press `u`: it comes back.
2. On a new line, press `i`, type `oops` and leave with `Esc`. Press `u` and the word disappears; press `ctrl+r` and it comes back.

## Your practice area

delete me

Next: [[B06 Save with a name and close]]
