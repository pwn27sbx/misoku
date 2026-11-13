# A03 · Editing properties

**You will learn:** to change a value from the preview's table and to know which properties are edited there and which only in the editor.

A single value is edited in the table with a click: a field opens with the current value, you change it and the click becomes one undoable change, as if you had typed it yourself. The key is not touched, so `trip` is still written `trip` even though the value changes.

Lists are different. A property like `tags: [beach, sun]` is drawn read-only: the table cannot know whether you want to add, remove or reorder, so that change is made in the editor, by typing the list by hand.

This difference matters: the table is for touching up a piece of data, the editor is for rewriting the structure.

## Your turn

1. Click the value `coast` in the table, change it to `mountains` and confirm.
2. Add `tags: [beach, sun]` to the front matter, try to edit the list from the table and, when you see it cannot be done, add `hiking` in the editor.



Next: [[A04 Note status]]
