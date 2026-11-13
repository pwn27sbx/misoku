# E18 · The global graph in depth

**You will learn:** the global graph keys you were missing.

The global graph (`space G`, `:graph all` or `g` on the home screen) draws the whole vault and its links. You move through it without a mouse: `h`, `j`, `k` and `l` move the selection between nodes, `n` and `N` jump to the next and previous one, `Enter` opens the node, `+` and `-` zoom in and out, `0` centres again and `Esc` clears the filter or goes back to the previous selection.

`o` hides orphan nodes and `u` the missing ones, those hollow circles for notes that do not exist yet. They are the two switches that turn a tangle into a readable map.

`/` filters the graph by note name: it does not search inside a note, it changes which nodes are drawn. And since `foo` would be an argument of the command, `:graph foo` is not a filter: it is a syntax error and the bar answers `E488: Trailing characters`. The filter is turned on with `/` inside the graph.

## Your turn

1. In the global graph use `h j k l`, `n`/`N`, `Enter`, `+`/`-`, `0`, `Esc`, `o` and `u`.
2. Press `/` and check that it filters the graph instead of searching inside a note.
3. Run `:graph foo` and read `E488: Trailing characters` in the bar.



Next: [[E19 Precision Vim]]
