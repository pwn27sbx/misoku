# E09 · Themes by name and your own themes

**You will learn:** to set a theme without opening the list and to add palettes of your own.

`:colo <name>` applies that theme and saves it in `config.json` in one go; you do not need to open the picker. It is the way to leave Misoku with a specific theme from the first start, and the one you can repeat on another machine. The names are the ones in the list: `onedark`, `catppuccin`, `zenburn`.

Your own themes are JSON files with a palette, and they live in the `themes/` folder next to `config.json` (on Linux `~/.config/misoku/themes/`). When you copy one there, it appears in the picker without restarting the app; if you edit the file, the change shows a few seconds later. Your own theme with the same name as a bundled one does not win: the bundled ones take precedence.

`themes_dir` swaps that folder for another, and accepts a folder full of JSON files or a single palette file. It is the setting for keeping your themes in a vault or in a separate repository.

## Your turn

1. Run `:colo` with a name, for example `:colo catppuccin`, and restart.
2. Copy a theme JSON into the `themes/` folder and watch it appear in the picker without restarting.
3. Change `themes_dir` to another folder and check that Misoku uses the themes from there.



Next: [[E10 Folders in config.json]]
