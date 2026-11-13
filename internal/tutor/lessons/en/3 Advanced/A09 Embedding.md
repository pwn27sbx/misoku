# A09 · Embedding

**You will learn:** to bring in the content of another note, of a section, of a block or of an image, without copying it.

A normal link takes you to another note. If you put an exclamation mark in front of it, the content comes here: `![[Trip]]` draws the whole note inside this one, and if `Trip` changes, this changes with it.

The pattern is always the same: `![[Note#part]]`. The `#` separates the note from the part, and `^` says the part is a block and not a heading. That is why `![[Trip#Luggage]]` brings the section under that heading and `![[Trip#^booking]]` brings only the line marked with `^booking`. In the same note the name is left out: `![[#^booking]]`. Without the `#`, `[[^booking]]` would be a note called `^booking`, which is why the `#` is not optional: it is the border between the note and the part.

An image works the same way: `![[mapa.png]]` draws it by its name, with no path. It is the way to have a photo or a plan inside the note without depending on where the file is. This lesson lives in `3 Advanced/` and `mapa.png` stays at the root of the tutorial: the link finds it anyway, because it searches by name, not by folder.

## Your turn

1. Create the note `Trip` with a heading `Luggage` and a line ending in `^booking`; then write `![[Trip]]` here on a new line.
2. Write `![[Trip#Luggage]]` and see that it brings only that section.
3. Write `![[Trip#^booking]]` and see that it brings only that block.
4. Write `![[mapa.png]]` with an image already in the vault, or with the name of the one you have.



Next: [[A10 File links and dangerous links]]
