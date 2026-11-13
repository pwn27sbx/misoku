# E11 · Sync in config.json

**You will learn:** which folders sync and what Misoku remembers about you.

`sync.vaults` is the list of vaults Misoku syncs. A folder that is not on the list never runs git: not on save, not on open, not on quit. That is why sync is opt-in, and why adding an entry is the step that turns it on; each entry can carry its `interval` and its `device`, the name that signs the commits and the conflict copies.

The way to add one by hand is one more entry in the object, with the path as the key: `{"sync": {"vaults": {"~/Notes": {}}}}`. The path has to be absolute or start with `~`, just like `notes_dir`. The convenient way is `:sync setup`, which writes it for you (lesson E17).

`learn_offered` is another small key: it stores `true` once you have answered the first-time dialog, whether "I'm new", "I have experience" or "Not now". That way it does not show again on that machine; if you delete the key, it asks once more.

## Your turn

1. Add a vault to `sync.vaults` and check that only that one syncs.
2. Look at `learn_offered` after answering the first-time dialog.



Next: [[E12 Syncing]]
