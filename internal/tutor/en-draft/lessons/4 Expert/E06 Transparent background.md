# E06 · Transparent background

**You will learn:** to let the terminal show behind the app and to draw an in-progress checkbox.

`ui.transparent` leaves the base backgrounds unpainted: editor, preview, explorer, tab line and status bar. What does get painted are highlights and floating boxes. You see the result with a terminal that has opacity or blur: the terminal's background shows behind Misoku.

The "Transparent background" row in `:settings` turns it on and off. Turning it off does not write `false`: it deletes the key, because off is the factory value. You can also write `"transparent": true` by hand inside `ui`, and the background changes on restart.

And while you are in `ui`, there is a setting you can only set by hand: `ui.checkbox.todo`. It is the glyph that draws an in-progress checkbox, the one written as `- [-]`. Without a glyph, `[-]` is plain text; with one, the preview paints the checkbox in yellow with your symbol. You have to restart Misoku to see it.

## Your turn

1. Turn on the Transparent background row and turn it off afterwards.
2. Check the same thing by writing `ui.transparent` by hand inside `ui`.
3. Add `ui.checkbox.todo` with the glyph you like and restart Misoku.
4. Write an in-progress task in this note, for example `- [-] call the hotel`, and look at it in the preview.



Next: [[E07 Numbers and cursor]]
