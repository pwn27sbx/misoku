# The README demo: docs/demo.gif, recorded with VHS (https://github.com/charmbracelet/vhs).
#
# Run it from the repository root:
#
#   vhs docs/demo.tape
#
# It builds misoku from this checkout and records it against a throwaway
# sample workspace (scripts/demo-workspace.sh) in a temporary folder. HOME and
# the XDG folders point into that folder too, so your own config, notes and
# cache are never read or changed, and no personal path appears on screen.

Output docs/demo.gif

Require go

Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 16
Set Width 1140
Set Height 740
Set Padding 20
Set WindowBar Colorful
Set BorderRadius 8
Set Theme "Catppuccin Mocha"
Set TypingSpeed 70ms

# Setup, off camera: build, create the sample notes, isolate HOME.
Hide
Type `export DEMO="$(mktemp -d)" && go build -o "$DEMO/misoku" ./cmd/misoku && sh scripts/demo-workspace.sh "$DEMO/Notes"`
Enter
Type `export PATH="$DEMO:$PATH" HOME="$DEMO" XDG_CONFIG_HOME="$DEMO/.config" XDG_CACHE_HOME="$DEMO/.cache" XDG_DATA_HOME="$DEMO/.local/share" MISOKU_SYNC=0 && cd "$DEMO/Notes" && clear`
Enter
Sleep 5s
Show

# Open the notebook: the start screen.
Type "misoku --theme catppuccin"
Sleep 300ms
Enter
Sleep 2s

# Find a note and open it: editor on the left, live preview on the right.
Type "f"
Sleep 400ms
Type "wel"
Sleep 900ms
Enter
Sleep 1.5s

# Move down through the blocks.
Type@120ms "jjjjjjjj"
Sleep 600ms

# Add a to-do with a wiki link, picked from the [[ suggestions.
Type "o"
Type "- [ ] Book the "
Type "[["
Sleep 500ms
Type "kyo"
Sleep 900ms
Enter
Sleep 300ms
Type " train"
Sleep 300ms
Escape
Sleep 1s

# Save.
Ctrl+S
Sleep 1s

# Follow the link: the other note shows where it is linked from.
Type@150ms "bbb"
Sleep 300ms
Type "gd"
Sleep 2.5s

# Reading mode: only the preview, full width.
Type " r"
Sleep 3s

# Quit and remove the sample workspace, off camera.
Hide
Ctrl+C
Sleep 1s
Type `cd / && rm -rf "${DEMO:?}"`
Enter
Sleep 500ms
