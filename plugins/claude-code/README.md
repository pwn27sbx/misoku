# Misoku rich page in a Claude Code pane (protocol)

`misoku --headless` renders a note as PNG tiles for a host that owns the
screen and the input. It is the same page the rich reading view draws
(`Space V`), from the same engine, and the contract mddog's Claude Code plugin
follows.

```sh
misoku --headless --dir ~/notes --note "Notes/Design.md" --cols 100 --rows 40
```

The command prints **one JSON line** and writes the tiles it names:

```json
{"protocol":1,"run":"4821-1732750000000000000","version":"0.37.0-dev","note":"Notes/Design.md",
 "theme":"onedark","cols":100,"rows":40,"page_rows":143,"width_px":1200,
 "tiles":[{"row":0,"cols":100,"rows":40,"width":1200,"height":960,
           "path":"/tmp/misoku-headless/4821-1732750000000000000/tile-0000.png"}]}
```

`run` names the folder this invocation owns: two panes or a re-render never
share files, and a tile is written to a temp name and renamed, so a host never
reads half a PNG.

Rules:

- `protocol` guards compatibility: a host that does not know the number must
  say which side to update instead of drawing.
- Tiles are full-pane PNGs, top to bottom, one per `rows` cells. Draw them
  with the Kitty graphics protocol behind Unicode placeholders (`U=1`), or
  write them to disk yourself; the pane keeps them until the host deletes
  them.
- The host owns scrolling and input; `page_rows` and each tile's `row` say
  where a tile belongs.
- Tiles live under the OS temp folder (`misoku-headless/`); delete the ones
  you no longer show.
- Nothing is written to the vault.

The plugin package itself (a `hooks/register` that runs the command and draws
its tiles) is not delivered here: it needs the Claude Code plugin API, which
belongs to that host. This README and the JSON line are the contract it must
follow.
