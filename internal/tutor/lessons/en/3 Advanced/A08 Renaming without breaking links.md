# A08 · Renaming without breaking links

**You will learn:** to rename a note without leaving broken links and to link the right note when two share a name.

Renaming in the explorer with `r` is not just changing the file. Misoku looks at which notes link to this one and asks you `Update N links in M notes?`. If you accept, it rewrites those links in all of them, in one step, and none is left pointing to a name that no longer exists. If you say no, the file changes and the links stay as they were.

Repeated names are the other side. With two notes called `Suitcase`, one in `Travel` and another in `Home`, the short link would be ambiguous. That is why, when you pick the one in a folder, the suggestion list writes the path: the link carries the folder inside it and points to exactly that note.

The two rules together are why linking with brackets comes cheap: the app keeps names up to date and disambiguates when it needs to.

## Your turn

1. Create a note called `Suitcase`, write `[[Suitcase]]` in this note, rename it with `r` to `Luggage` and accept updating the links.
2. Create `Home` and `Travel` with a note `Suitcase` inside each, link the one in `Home` and see that the link carries the folder in its path.



Next: [[A09 Embedding]]
