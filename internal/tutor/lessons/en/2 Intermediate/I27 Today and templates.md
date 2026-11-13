# I27 · Today and templates

**You will learn:** to write the note of the day and to reuse templates with data.

The daily note is named after the date and lives in `Diario/`, the daily notes folder: `space o d` (or `:daily`) opens today's, and the first save creates it. A template is a normal note in `Plantillas/`, the templates folder, that you insert with `/Template`; whatever it holds lands at the cursor. Inside the template, `{{date}}` becomes today's date and `{{title}}` the name of the note you insert it into. Properties are the most useful part: the keys the note already has are kept and only the missing ones are added, so a template can bring `place:` without overwriting your `stage:`.

For this lesson, add a properties block at the top of the note with `stage: in progress`.

## Your turn

1. Press `space o d` (or `:daily`) and save the daily note.
2. Create the note `Plantillas/daily` with two lines of text: `Day plan` and `No rush`.
3. In this note, insert `/Template` and pick `daily`.
4. Add two lines to the template, `Date: {{date}}` and `Note: {{title}}`, and insert it here again.
5. Add `stage: new` and `place: beach` to the template's properties and insert it into this note.

Next: [[I28 The self-folding explorer]]
