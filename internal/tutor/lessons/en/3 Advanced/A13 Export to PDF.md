# A13 · Export to PDF

**You will learn:** to turn a note into a PDF with the size of a page, in the folder you want and with the name you want.

`:export` takes the current note, draws it like the preview but paginated, and writes the PDF next to the file. If you say nothing else, the name is the note's. What does not fit on one page continues on the next, and tables and lists break where they should.

With an argument you change the target. A folder, ending in a slash, puts the PDF inside it with the note's name; a file name writes it next to the note with that name. `:pdf` is the same command.

`space P` does the same without typing anything, and it works from any open note. It is the quick way to email a note or print it.

## Your turn

1. Run `:export` in this note and see the PDF that appears next to it.
2. Run `:export ~/Documents/` (or a folder of yours) and then `:export report.pdf`.
3. Open the note `A00 Advanced` and press `space P`.



Next: [[A14 Cover]]
