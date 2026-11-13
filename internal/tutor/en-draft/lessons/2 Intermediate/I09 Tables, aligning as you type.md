# I09 · Tables: aligning as you type

**You will learn:** to keep the table tidy while you type, without counting spaces.

A table reads well when all the bars fall in the same column. `:tablefmt` (or `:tf`) pads the cells to get there in one go. And when you type inside a padded cell, Misoku uses up that padding: the closing bar does not move while the text fits, and when it no longer fits, the column widens in every row at once. The preview draws the table with borders, without the Markdown dashes.

## Your turn

1. Write a two-column table with texts of different lengths, put the cursor inside and run `:tablefmt` (or `:tf`).
2. Type `Beach` inside a padded cell: the closing bar does not move while it fits, and once it overflows the column widens.
3. Look at the table drawn with borders in the preview.

Next: [[I10 Tables from the preview]]
