# E10 · Folders in config.json

**You will learn:** the three folders Misoku reads from the configuration.

`notes_dir` is the folder Misoku opens when you launch it with no arguments, just `misoku`. It has to be an absolute path or start with `~`, because a relative path would depend on the folder you start from. Changing it is what makes your desktop icon open your vault and not another one.

`daily_dir` and `templates_dir` are folders inside the vault, not outside it. `daily_dir` decides where daily notes are written (`YYYY-MM-DD.md`) and its factory value is `Diario`; `templates_dir` decides where `/Template` looks for templates, with `Plantillas` by default. Misoku rejects an absolute path, one that leaves the vault, or the vault's root, and tells you why: that way a typo does not write notes outside your notes.

Both are written the same way: a relative path, separated by `/`.

## Your turn

1. Change `notes_dir` to another folder and open Misoku with no arguments.
2. Set `daily_dir` to a folder inside the vault, for example `Diario/Trips`; a folder outside is not accepted.
3. Set `templates_dir` to `Plantillas` and check that `/Template` lists its notes.



Next: [[E11 Sync in config.json]]
