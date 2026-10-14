# Contributing to Misoku

Thanks for wanting to help! Misoku is a terminal notebook that edits plain
Markdown files, so every contribution should keep two promises:

1. **Notes stay plain Markdown.** Anything Misoku adds (views, answers,
   covers) must be readable and editable as text in any other editor.
2. **Nothing is lost by accident.** Unsaved work is never discarded without
   asking, and files are only written when the user saves.

## Ways to contribute

- **Report a bug:** [open an issue](https://github.com/pwn27sbx/misoku/issues/new)
  with your OS, terminal, `misoku --version`, what you did, what you
  expected and what happened. A screenshot or the note that triggers it
  helps a lot. The debug log lives in your cache folder
  (`~/Library/Caches/misoku/misoku.log` on macOS,
  `~/.cache/misoku/misoku.log` on Linux,
  `%LOCALAPPDATA%\misoku\misoku.log` on Windows).
- **Suggest a feature:** open an issue describing the problem it solves
  before writing a large change, so we can agree on the approach first.
- **Send a fix or feature:** small, focused pull requests are easiest to
  review and merge.
- **Docs, themes and translations of the README** are welcome too.

## Development setup

You need the Go version listed in [`go.mod`](go.mod) and, ideally, a
[Nerd Font](https://www.nerdfonts.com/) in your terminal.

```sh
git clone https://github.com/pwn27sbx/misoku.git
cd misoku
make run      # run from source (opens the current folder as the workspace)
make test     # run the test suite
make          # fmt + vet + test + build into ./bin/misoku
```

Try your change against a scratch notes folder rather than the repository
itself:

```sh
go run ./cmd/misoku --dir /tmp/misoku-notes
```

## Project layout

| Package | What lives there |
| :-- | :-- |
| `cmd/misoku` | Entry point, flags, config loading |
| `internal/app` | Root model: layout, keys, mouse, tabs, dialogs, view actions |
| `internal/content` | Editor pane, `/` menu, table editing |
| `internal/preview` | Markdown renderer, images, board/calendar/timeline/chart/form views |
| `internal/sidebar` | Explorer tree |
| `internal/dashboard` | Start screen |
| `internal/core` | Filesystem store and messages (no UI code) |
| `internal/highlight` | Syntax highlighting |
| `internal/textarea` | Vendored bubbles textarea (changes marked `misoku:`) |
| `internal/theme` | Theme palettes (`palettes_gen.go` is generated) |
| `internal/ui` | Exact-size layout helpers and overlays |
| `internal/config` | Saved preferences |

## Guidelines

**Code**

- Run `make` before pushing: code must be `gofmt`-formatted, pass
  `go vet` and all tests.
- Keep the build pure Go (`CGO_ENABLED=0`). Misoku ships as one static
  binary for macOS, Windows and every Linux distro, so no cgo or system
  libraries.
- Match the surrounding style: small focused functions, comments that
  explain *why*, no dead code.
- New dependencies need a good reason and a permissive license (MIT, BSD,
  Apache-2.0).

**Behaviour**

- Every pane must render at exactly its given width and height (use
  `ui.Fit` / `ui.FitLine`); a single extra cell breaks the dividers.
- Keyboard first, mouse friendly: features should work from the keyboard
  and, where it makes sense, by clicking.
- Think about all three platforms: paths via `path/filepath`, note IDs with
  `/`, no shell commands that only exist on one OS (or provide a fallback,
  like the built-in file browser).
- When adding or changing a key binding, update the in-app cheatsheet in
  `internal/app/modal.go` (the README stays short and doesn't list keys).

**Tests**

- Add a test for every bug fix (one that fails without the fix) and for
  new behaviour.
- UI flows are tested by driving the real model: see the `harness` in
  `internal/app/app_test.go` (send keys and mouse clicks, then check the
  rendered frame and the saved file).
- Tests must not touch the user's real clipboard, config or home folder:
  use `app.WithClipboard`, `t.TempDir()` and `t.Setenv`.

**Commits and pull requests**

- Use [Conventional Commits](https://www.conventionalcommits.org/):
  `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `ci:`, `build:`,
  `chore:`.
- Stage files by name and check `git status` before committing; it's easy
  to commit test notes or images by accident.
- One topic per pull request, with a short description of what changed and
  how you tested it (screenshots for UI changes).
- CI runs the tests on Linux, macOS and Windows and test-installs a build
  on several Linux distributions; please make sure it's green.

## Releases

Maintainers release by pushing a version tag (`git tag v0.2.0 && git push
origin v0.2.0`). GoReleaser builds the binaries, Linux packages and
checksums and publishes them on GitHub.

## Code of conduct

Be kind and assume good intent. Harassment or personal attacks are not
welcome in issues, pull requests or discussions; maintainers may remove
content or block people who don't respect that.

## License

By contributing you agree that your contributions are licensed under the
[MIT License](LICENSE).
