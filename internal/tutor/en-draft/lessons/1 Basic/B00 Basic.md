# Basic route

Welcome to the Misoku tutorial. This route assumes nothing: you start with the first lesson open and you are typing within thirty seconds.

Each route lives in its own folder in the explorer (`1 Basic`, `2 Intermediate`, `3 Advanced`, `4 Expert`), so its lessons stay together and in the order you learn them. `After the tutorial.md` and `mapa.png` are at the root of the tutorial.

If you came from the first-run dialog, this is what you chose. **I'm new** opens this route; **I have experience** takes you straight to Intermediate; **Not now** leaves you in your notes. That dialog only decides where you start, and you can switch routes whenever you like from the **Learn** entry on the home screen.

## How to play

Each lesson is a note like this one. At the top it explains what you will learn and why; below, under **Your turn**, are the numbered tasks. Do them in the note itself: Misoku watches what you type or the action you take and tells you in the status bar.

The bar shows the lesson and the current step, with what is left to do (`B03 · Step 2/4 · …`). When a step is done you see `✓ Step 2`; when you finish the lesson, `✓ B03 passed` with the name of the next one; and when you finish the route, `🏆 Basic route complete`.

Examples are always between backticks: `i`, `ctrl+s`, `[[Note]]`. The checker ignores everything in code, so copying an example doesn't count; type it yourself on a new line.

## The two modes

Misoku uses vim's modes. In NORMAL every letter is a command; to type you enter INSERT with `i` (or `a`, `o`…) and leave with `Esc`. If something doesn't respond the way you expect, check the mode: almost always you are still in INSERT.

## The 26 lessons

1. [[B01 Write in 30 seconds]] — `i`, `Esc` and `ctrl+s`
2. [[B02 Insert anywhere]] — `a`, `A`, `I`, `o`, `O`
3. [[B03 Delete without fear]] — `x` and `dd`
4. [[B04 Move without a mouse]] — `hjkl`, arrows and jumps
5. [[B05 Undo and redo]] — `u` and `ctrl+r`
6. [[B06 Save with a name and close]] — `:w` and `space x`
7. [[B07 Tabs]] — `H`, `L`, `]`, `[`
8. [[B08 The explorer and focus]] — `space e` and focus
9. [[B09 Open and create notes]] — the tree and `n`
10. [[B10 Rename and delete]] — `r` and `d`
11. [[B11 Fold and peek]] — `l`, `h`, `z`, `W`, `p`
12. [[B12 The slash menu]] — the menu that writes Markdown
13. [[B13 Headings]] — from `#` to `####`
14. [[B14 Lists]] — bullets, numbers and tasks
15. [[B15 Checkboxes, check off]] — three ways to check off
16. [[B16 Checkboxes, remove and click]] — remove the checkbox and click
17. [[B17 Emphasis]] — bold, italics, code and strikethrough
18. [[B18 Quotes and divider]] — `>` and `---`
19. [[B19 Your first linked note]] — `[[`
20. [[B20 Follow a link]] — `gd`, `gx`, `Enter`, the mouse and `ctrl+o` to go back
21. [[B21 Read without editing]] — reading view and editor view: `space p` and `space r`
22. [[B22 Search inside the note]] — `/`, `n`, `N`
23. [[B23 Paste and make mistakes]] — `ctrl+v` and errors
24. [[B24 The Misoku home screen]] — `space d`
25. [[B25 Quit and come back]] — `:q`, `:wq`, `:q!`
26. [[B26 Basic challenge]] — a real shopping list

When you want to stop, `ctrl+c` closes Misoku and your progress is saved step by step: when you run `misoku tutor` again you pick up where you left off. To start over, `misoku tutor --reset` sends the tutorial to the trash.

Next: [[B01 Write in 30 seconds]]
