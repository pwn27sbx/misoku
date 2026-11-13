# B03 · Delete without fear

**You will learn:** to delete a character with `x` and a whole line with `dd`.

`x` deletes the letter under the cursor, without entering INSERT. `dd` is your first operator: `d` means "delete" and the second `d` says "the line". Vim combines them, and that idea (operator plus object) is the one you will use for life: `dw` deletes a word and `d$` deletes to the end of the line, although today you only need `dd`.

## Your turn

1. In the practice area, put the cursor on the second `t` of `hottel` and press `x`.
2. Put the cursor on the line `draft` and press `dd`: the whole line disappears.

## Your practice area

hottel

draft

Next: [[B04 Move without a mouse]]
