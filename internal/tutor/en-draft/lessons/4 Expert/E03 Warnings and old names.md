# E03 · Warnings and old names

**You will learn:** what Misoku warns about when a shortcut clashes, and why the old names still work.

A badly placed shortcut breaks nothing: Misoku checks it at startup and warns you. There are three typical clashes: the same key on two actions, an explorer key that never arrives because another action catches it first, and a `space o` that is the start of a longer sequence. The warning shows at startup and stays in `:messages` so you can reread it at your own pace.

The action names changed in version 0.27: `reading_mode` is now `reading_view` and `toggle_preview` is `editor_view`. The old names are still accepted and do the same thing, so an old configuration does not lose its shortcuts when you update.

The right thing is to use the new names; the old ones are there only so that what you already had does not break.

## Your turn

1. Give the same key to two actions, restart and read the warning at startup and in `:messages`.
2. In `keys`, use the old names `reading_mode` and `toggle_preview` and check that they turn on the reading view and the editor.



Next: [[E04 The settings menu]]
