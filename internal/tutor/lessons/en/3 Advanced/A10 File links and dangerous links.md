# A10 · File links and dangerous links

**You will learn:** to link a file in the vault and to recognise when a link deserves a question before it opens.

A normal Markdown link can point to any file, not just a note: `[menu](menu.pdf)` hands it to your system so it opens it with the right program. It is the way to keep a PDF, a spreadsheet or an image one click away from the note.

When the target is not so innocent, Misoku asks. A scheme like `file:///`, a path that leaves the vault or an executable file opens a dialog with Cancel already selected, so that one click too many does not launch anything.

The dialog is the rule: if the target is not clear, it asks first, and the safe answer is the one already marked. Accepting is always a decision, never an accident.

## Your turn

1. Write `[menu](menu.pdf)`, or the link to a file already in the vault, and follow it: it opens with the system's program.
2. Write `[odd](file:///etc/hosts)`, try to follow the link and check that the dialog asks for permission with Cancel selected.



Next: [[A11 Local graph]]
