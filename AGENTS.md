# Code Review Rules

Rules for reviewing changes to Misoku, a terminal Markdown notebook written in
Go (Bubble Tea). They are read by Gentleman Guardian Angel (`gga`) on every
commit, and are a good starting point for any AI agent working on this repo.

**Judge only what the commit changes.** The reviewer receives whole files, but
the rules apply to new or modified code; existing code elsewhere in a file is
not a reason to fail the commit.

## General

REJECT if:
- Hardcoded secrets, credentials or tokens, or personal paths such as `/Users/<name>`
- Debug output left in production code (`fmt.Println`, `log.Print` used for debugging)
- Duplicated logic that already exists elsewhere in the package
- A behaviour change without a test that pins it

## Go

REJECT if:
- Code not formatted with `gofmt`
- Exported identifiers without a doc comment
- Errors silently dropped where they matter: a failed write, rename, git command
  or other operation the user depends on must be returned or reported.
  Ignoring an error is acceptable only for best-effort cleanup (closing a file,
  removing a temporary file) and must be obvious from the context or a comment.
- Naked returns in functions longer than a few lines
- Running a background external program (for example `git` for sync) without a
  context or time limit. Interactive helpers the user drives (file pickers) and
  openers started with `Start()` (browser, file manager) are fine without one.
- Blocking work (disk walks, network, external programs) inside the Bubble Tea
  `Update` loop instead of a `tea.Cmd`

PREFER:
- Small, focused functions and table-driven tests
- Tests through the public interface of a package
- Comments that explain *why*, not *what*

## Data safety

REJECT if:
- A change can lose or overwrite a user's notes without asking (saves, deletes,
  renames, sync)
- An existing note overwritten in place instead of through the atomic save in
  `internal/core` (creating a new, empty file directly is fine)
- Deleting notes for good on macOS or Linux instead of moving them to the Trash

## Portability

REJECT if:
- Code that only works on one OS without a build tag or a runtime check:
  Misoku ships for macOS, Linux and Windows and CI tests all three

## Response Format

FIRST LINE must be exactly:
STATUS: PASSED
or
STATUS: FAILED

If FAILED, list: `file:line - rule violated - issue`
