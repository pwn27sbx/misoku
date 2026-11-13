# E17 · Setting up sync

**You will learn:** to go from nothing to a synced private repository.

`:sync setup`, or `s` on the home screen, is the wizard that sets everything up: it creates a private repository in your GitHub account, prepares the git identity, runs `gh auth setup-git` so the credentials are stored, and adds the vault to `sync.vaults`. When it finishes, the vault has `.git`, has `origin` and shows up in the list that syncs.

If you already created the repository and prefer the terminal, `misoku sync init <url>` makes the link without the wizard and handles three cases: first computer (empty repository, it uploads your notes), another computer (repository with notes, it downloads them) and both at once (notes on both sides: it merges and keeps everything). The third case is the one that creates conflict copies instead of choosing for you.

Both paths end in the same place: a vault in `sync.vaults` with its remote. The wizard is for the first time; the command, for repeating it on another machine.

## Your turn

1. Run `:sync setup` (or `s` on the home screen) and answer its questions.
2. Run `misoku sync init <url>` in a new folder and check that the repository is created and the vault is listed.



Next: [[E18 The global graph in depth]]
