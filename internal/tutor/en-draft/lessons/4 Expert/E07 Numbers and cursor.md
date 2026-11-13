# E07 · Numbers and cursor

**You will learn:** to number headings and to change the cursor's shape.

"Heading numbers" numbers the preview's headings like a table of contents: the first level-one heading is `1`, its children `1.1` and `1.2`, the next one `2`. It replaces the level icons, so if you had Numbers in Icons & bullets, you will see the numbering in its place. Without the key, numbering is off and each heading shows only its text; turning it on writes `"heading_numbers": true` inside `ui` and turning it off leaves it at `false`.

`ui.cursor_shape: auto` makes the drawn cursor follow the mode, as in Neovim: a block in NORMAL and VISUAL, a thin bar in INSERT. With `block`, the cursor is always a block. The environment variable `MISOKU_CURSOR_SHAPE`, if set, wins over the file; it is one of those things you set before opening the app.

With `auto` you always have a hint of which mode you are in, without looking at the status bar.

## Your turn

1. Turn "Heading numbers" on and off and leave it off so the change is saved.
2. With `ui.cursor_shape: auto`, look at the block in NORMAL and the bar when you enter INSERT.



Next: [[E08 The theme picker]]
