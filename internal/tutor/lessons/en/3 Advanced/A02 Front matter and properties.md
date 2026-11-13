# A02 · Front matter and properties

**You will learn:** to declare a note's data in its header and to read it as a table in the preview.

The front matter is a block at the start of the file, between two lines of three dashes, with key and value pairs. Misoku reads it as simple YAML: each key is a property of the note, not text to be read in the body.

The useful part comes next: the preview turns those properties into a table under the header, with one row per key. That way a trip note shows its destination and its stage without opening the header.

`cover` and `icon` are the exception. They are presentation properties, drawn in the cover and in the explorer, and the table does not show them so they are not repeated.

## Your turn

1. At the start of this note, above the title, write a block with `---`, `trip: coast`, `stage: planned` and another `---`.
2. Look at the properties table under the preview's header.
3. Check in that table that neither `cover` nor `icon` appears.



Next: [[A03 Editing properties]]
