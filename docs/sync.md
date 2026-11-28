# Git Sync

Misoku can keep a notes folder (*vault*) synchronized across your computers using your own private git repository (GitHub, GitLab, Gitea, or self-hosted). There is no Misoku server, and notes remain plain files with full git history.

## How It Works

Sync is off by default and only runs for folders explicitly listed in `sync.vaults` in `config.json`. For a listed vault, Misoku:

- commits changes locally shortly after you save (no network required);
- fetches, merges, and pushes on open, every 5 minutes (configurable), on `:sync` / `space s`, and on quit (up to 10 s timeout).

Nothing runs in the background while Misoku is closed. Set `MISOKU_SYNC=0` to disable sync for a single run.

## Set Up with GitHub (Recommended)

Requires git 2.38+ and `gh` (installed alongside `misoku` by the install scripts, or via your package manager).

Press `s` on the start screen, run `:sync setup` inside Misoku, or run:

```sh
misoku sync setup --dir ~/Notes
```

The assistant signs in via `gh auth login`, configures git credentials (`gh auth setup-git`), creates or selects a **private** repository (refusing public repos and code repositories), and adds the folder to `sync.vaults`. On another computer, choose **Bring my notes from another computer** on the first-run welcome screen.

Misoku never stores or prints your GitHub token; `gh` keeps it in your system keychain.

## Set Up with Any Git URL (SSH, GitLab, Gitea)

```sh
misoku sync init git@github.com:you/notes.git --dir ~/Notes
```

This creates/connects the repository, adds `* -text` to `.gitattributes`, excludes Misoku temp files in `.git/info/exclude`, and registers the vault in `sync.vaults`.

## Configuration

```json
{
  "sync": {
    "vaults": {
      "~/Notes": { "device": "laptop", "interval": "10m" }
    }
  }
}
```

- `device` (optional): name used in commit messages and conflict copies (defaults to hostname).
- `interval` (optional): timed sync interval (`"5m"` default, minimum `"1m"`, or `"off"` to sync only on open/manual/quit).

## Privacy & Conflicts

- **Keep your repository private:** notes are stored unencrypted in git.
- **No conflict markers:** if two devices edit the same lines, your version stays in `plan.md` and the other version is saved beside it as `plan (conflict laptop 2026-10-15 1430).md`. Compare, merge what you want, and delete the conflict copy.
- **Unsaved edits are protected:** an open note with unsaved edits is never overwritten on reload; saving keeps both versions.

## What Does Not Sync

- Files larger than **50 MB** (to stay under GitHub's 100 MB limit)
- Files matched by `.gitignore`
- Nested git repositories inside the vault
- Targets of symlinks pointing outside the vault
- New filenames invalid on Windows (`:`, `*`, `?`, `CON`, `NUL`, trailing dot/space)

## Troubleshooting

Run `:sync status` (or press `space s` when the status bar shows a sync pause) for details:

| Status / Error | What to do |
| --- | --- |
| `offline · N pending` | Commits are saved locally and push automatically once back online. |
| `sync paused, lock left · space s` | If no git process is running, press `space s` and choose **Remove the lock and sync**. |
| `GitHub did not let this computer in` | Press `space s` to sign in again with `gh`. |
| `origin's SSH host key is not known yet` | Run `misoku sync init <url>` once in a terminal to accept the host key. |
| `… was rewritten (a force-push)` | Sync pauses to avoid re-uploading deleted data; reconcile with `git fetch origin` and `git log origin/main`. |
| `another Misoku window syncs this vault` | Only one window syncs a vault at a time; the next window takes over when the first closes. |
