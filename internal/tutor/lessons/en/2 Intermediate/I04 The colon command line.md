# I04 · The `:` command line

**You will learn:** to ask for things by name with `:`, including opening a specific note.

A colon opens a command line in a box near the top of the screen, titled Cmdline. `Enter` runs it and `Esc` leaves it. That is where the commands that don't fit on a key live: `:w` saves without closing anything, and `:e` opens a note by its name, without going through the explorer. `:e` accepts three forms: a name, a name with a heading after `#`, and a path with folders separated by `/`. If the note does not exist, it opens a draft with that name for you to write.

Prepare two notes before you start: `Trip`, with a heading `Packing`, and `Projects/Home`, inside a folder `Projects`.

## Your turn

1. Press `:` in NORMAL and leave it with `Esc`.
2. Run `:w` and see that it saves without leaving.
3. Run `:e Trip` to open the note `Trip`.
4. Try `:e Trip#Packing`, `:e Projects/Home` and finally `:e Beach`, which does not exist and opens a draft.

Next: [[I05 Collapsible blocks and callouts]]
