# E19 · Precision Vim

**You will learn:** two details almost nobody uses that you will be glad to have.

`ctrl+e` and `ctrl+y` scroll the view one line down or up without moving the cursor. They let you look at what is further down without losing the spot where you were writing: the cursor stays on its line and the text slides underneath. It is like peeking over the edge of the page.

The undo history is per tab, not per application. Each note has its own stack, with up to 200 changes or 8 MB in memory, and a new tab starts empty. That is why, if you edit in one note, switch to another and press `u`, you do not undo what you did in the first one: there is nothing to undo in the second, and the bar says `Already at oldest change`.

The consequence: you can work on two notes at once without their histories getting mixed. And if you reload a note, its stack starts from zero.

## Your turn

1. Scroll one line with `ctrl+e` and `ctrl+y` without moving the cursor.
2. Edit in one tab, switch to another and press `u`: the second one's history is empty.



Next: [[E20 Expert Challenge]]
