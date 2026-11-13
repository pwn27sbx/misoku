# After the tutorial

This note gathers what is not practised inside the routes: the terminal commands and the features that depend on other programs. It is for reference: it has no steps.

## Terminal commands

- `misoku tutor` opens the tutorial and leaves you at the first step you have not passed.
- `misoku tutor --route I` opens a specific route (`B`, `I`, `A` or `E`).
- `misoku tutor --reset` moves the tutorial to the trash and starts from zero.
- `misoku <folder>` opens another folder, and `misoku --dir <folder>` does the same from anywhere.
- `misoku export note.md -o output.pdf` writes a note as a PDF, as does `:export` inside the app.
- `misoku paste-image` saves an image from the clipboard into the note from outside.
- `misoku sync setup` connects the vault to GitHub and `misoku sync init` prepares it by hand.
- `misoku --version` gives the version, `misoku --help` lists the options, `misoku --list-themes` the themes and `misoku --theme <name>` picks one.
- `misoku --themes-dir <folder>` reads your own themes, and the `MISOKU_WORKSPACE` variable sets the folder that opens.

## What is practised outside

Sync with git, its conflicts and the arrival notices need two computers or a real remote; the tutorial does not invent one. The same goes for PDF export, the system's file dialog, environment variables and the "terminal too small" frame.

## What you only see

Some things are only visual: the dot on a tab with unsaved changes, the word counter, `Ln N, Col M`, the focus colours and the preview's syntax highlighting. You learn them by looking, not with a step.

## Keep learning

The Intermediate, Advanced and Expert routes cover the rest: motions and selection, the command line, tables, tags, properties, daily notes, history, graph, templates, sync and every setting. Each route assumes the one before, but you can go straight to the one you need from the home screen or with `misoku tutor --route`.
