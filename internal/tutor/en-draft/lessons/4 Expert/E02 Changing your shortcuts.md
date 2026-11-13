# E02 · Changing your shortcuts

**You will learn:** to reassign one key and unassign another.

All shortcuts live in `keys`, inside `config.json`. Each action keeps the list of keys that trigger it, so you can add a key to an action, take one away or leave it with none. The action's name is the one you see in the command palette.

An empty list is the way to say "this action has no shortcut": `[]`. The action still exists in the palette and can be run from there, but no key triggers it. It is useful when a key gets in your way and you would rather run the action by hand.

Changes are read at startup. Save the file, close Misoku and open it again to try the new key.

## Your turn

1. In `config.json`, inside `keys`, give `reading_view` another key, for example `z`: `"reading_view": ["space p", "z"]`. Restart and check that `z` opens the reading view.
2. Give another action, for example `export_pdf`, the value `[]` and check that it stops responding to its key even though it is still in the palette.

If the JSON has an extra comma, Misoku starts with the factory shortcuts and warns you in `:messages`; the next lesson is about exactly those warnings.



Next: [[E03 Warnings and old names]]
