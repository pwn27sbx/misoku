# A05 · Advanced filters

**You will learn:** to combine status, folder and a word in a single search, and to reach that filter from the `:status` command.

The finder understands conditions as well as words. `status:active` keeps only the active notes, and `folder:Travel` keeps only the ones that live in that folder, anywhere along their path. A folder with spaces is written in quotes: `folder:"My notes"`.

The `:status` command is a shortcut: it opens the finder already filtered by a status, or offers you the four to choose from. It is the quick way to see the day's board.

The good part comes when you combine them: every condition must hold and, on top of that, the word has to appear in the note's name or text. `status:active folder:Travel beach` answers a very specific question without opening a single note.

## Your turn

1. Run `:status` and choose one of the four statuses.
2. Open the finder and type `status:active`.
3. Type `folder:Travel` and then `folder:"My notes"`.
4. Combine the three things in one search: `status:active folder:Travel beach`.



Next: [[A06 Note history]]
