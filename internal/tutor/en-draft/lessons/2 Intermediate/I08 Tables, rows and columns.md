# I08 · Tables: rows and columns

**You will learn:** to add and remove rows and columns with four commands.

Misoku tables are not edited by hand as they grow: there are commands that know where the cursor is and touch the right row or column. `:addrow` (or `:tr`) adds a row below; `:addcol` (or `:tc`) adds a column to the right; `:delrow` and `:delcol` remove them. All of them keep the alignment, so there is no need to move the dashes around by hand.

## Your turn

1. Insert `/Table`, put the cursor inside and run `:addrow` (or `:tr`).
2. Run `:addcol` (or `:tc`): a new column appears.
3. Run `:delrow`: the table loses a row.
4. Run `:delcol`: every row loses a cell.

Next: [[I09 Tables, aligning as you type]]
