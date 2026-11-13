package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pwn27sbx/misoku/internal/clipimage"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"

	tea "github.com/charmbracelet/bubbletea"
)

// mediaKind describes what the "/" menu asked to attach.
type mediaKind struct {
	name  string   // image, video, audio, file: an ID, never shown
	noun  i18n.Key // what it is, shown: "image"
	title i18n.Key // dialog title
	exts  []string // allowed extensions (empty = any file)
	uti   string   // macOS uniform type for the native dialog
}

var mediaKinds = map[string]mediaKind{
	"image": {"image", i18n.AppKindImage, i18n.AppChooseImage, core.ImageExts, "public.image"},
	"video": {"video", i18n.AppKindVideo, i18n.AppChooseVideo, core.VideoExts, "public.movie"},
	"audio": {"audio", i18n.AppKindAudio, i18n.AppChooseAudio, core.AudioExts, "public.audio"},
	"file":  {"file", i18n.AppKindFile, i18n.AppChooseFile, nil, ""},
	"cover": {"cover", i18n.AppKindCover, i18n.AppChooseCover, []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"}, "public.image"},
}

// windowsDialogScript is the PowerShell that shows Windows' file dialog for
// k, titled and filtered in the language in use.
func windowsDialogScript(k mediaKind) string {
	filter := i18n.T(i18n.AppAllFiles) + "|*.*"
	if len(k.exts) > 0 {
		filter = i18n.T(k.noun) + "|*" + strings.Join(k.exts, ";*")
	}
	return fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; $d = New-Object System.Windows.Forms.OpenFileDialog; $d.Title = %s; $d.Filter = %s; if ($d.ShowDialog() -eq 'OK') { $d.FileName }`, psQuote(k.titleText()), psQuote(filter))
}

// psQuote is s as a PowerShell single-quoted string: a quote inside is
// doubled, so no title or filter can end the string early.
func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// titleText is the dialog's title in the language in use.
func (k mediaKind) titleText() string { return i18n.T(k.title) }

func (k mediaKind) accepts(name string) bool {
	if len(k.exts) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	for _, e := range k.exts {
		if e == ext {
			return true
		}
	}
	return false
}

// mediaPickedMsg reports the result of the native file dialog.
type mediaPickedMsg struct {
	kind        string
	path        string
	cancelled   bool
	unavailable bool // no GUI dialog could be shown: use the built-in browser
	err         error
}

var errNoDialog = errors.New("no native file dialog")

// pickMedia opens the operating system's file dialog in the background; if
// none is available (SSH, no desktop, MISOKU_NATIVE_PICKER=0) the built-in
// browser takes over.
func (m *Model) pickMedia(kind string) tea.Cmd {
	k, ok := mediaKinds[kind]
	if !ok {
		return nil
	}
	if os.Getenv("MISOKU_NATIVE_PICKER") == "0" {
		return m.openFileBrowser(kind)
	}
	m.setStatus(i18n.T(i18n.AppDialogOpened, k.titleText()))
	return func() tea.Msg {
		p, cancelled, err := nativeFileDialog(k)
		if errors.Is(err, errNoDialog) {
			return mediaPickedMsg{kind: kind, unavailable: true}
		}
		return mediaPickedMsg{kind: kind, path: p, cancelled: cancelled, err: err}
	}
}

func nativeFileDialog(k mediaKind) (string, bool, error) {
	switch runtime.GOOS {
	case "darwin":
		if os.Getenv("SSH_CONNECTION") != "" {
			return "", false, errNoDialog
		}
		script := fmt.Sprintf(`POSIX path of (choose file with prompt %q`, k.titleText())
		if k.uti != "" {
			script += fmt.Sprintf(` of type {%q}`, k.uti)
		}
		script += ")"
		out, err := exec.Command("osascript", "-e", script).Output()
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) && strings.Contains(string(ee.Stderr), "-128") {
				return "", true, nil // user pressed Cancel
			}
			return "", false, errNoDialog
		}
		return strings.TrimSpace(string(out)), false, nil

	case "linux", "freebsd", "openbsd":
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			return "", false, errNoDialog
		}
		var cmd *exec.Cmd
		if bin, err := exec.LookPath("zenity"); err == nil {
			args := []string{"--file-selection", "--title=" + k.titleText()}
			if len(k.exts) > 0 {
				args = append(args, "--file-filter="+i18n.T(k.noun)+" | *"+strings.Join(k.exts, " *"))
			}
			cmd = exec.Command(bin, args...)
		} else if bin, err := exec.LookPath("kdialog"); err == nil {
			filter := ""
			if len(k.exts) > 0 {
				filter = "*" + strings.Join(k.exts, " *")
			}
			cmd = exec.Command(bin, "--getopenfilename", ".", filter, "--title", k.titleText())
		} else {
			return "", false, errNoDialog
		}
		out, err := cmd.Output()
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) && ee.ExitCode() == 1 {
				return "", true, nil
			}
			return "", false, errNoDialog
		}
		return strings.TrimSpace(string(out)), false, nil

	case "windows":
		out, err := exec.Command("powershell", "-NoProfile", "-STA", "-Command", windowsDialogScript(k)).Output()
		if err != nil {
			return "", false, errNoDialog
		}
		p := strings.TrimSpace(string(out))
		return p, p == "", nil
	}
	return "", false, errNoDialog
}

// attachMedia copies the picked file next to the note and links it.
func (m *Model) attachMedia(kind, src string) {
	link := m.attachMediaAs(kind, src, false)
	if link != "" && kind != "cover" { // the cover sets its own confirmation
		m.setStatus(i18n.T(i18n.AppAttached, link))
	}
}

// attachPastedImage saves a picture read from the clipboard next to the note and
// links it on a line of its own, so the preview draws it.
func (m *Model) attachPastedImage(msg core.ClipboardImageMsg) {
	defer clipimage.Discard(clipimage.Data{Path: msg.Path})
	link := m.attachMediaAs("image", msg.Path, true)
	if link == "" {
		return // the status line already says why nothing was attached
	}
	m.setStatus(i18n.T(i18n.AppPasted, link))
}

// attachMediaAs is attachMedia with the choice of putting the link on a line of
// its own, which a picture needs for the preview to draw it instead of an icon.
// It returns the link it inserted, or "" when nothing was inserted.
func (m *Model) attachMediaAs(kind, src string, ownLine bool) string {
	b := m.activeBuffer()
	if b == nil {
		m.setStatus(i18n.T(i18n.AppOpenNoteFirst))
		return ""
	}
	noteDir := b.dir
	if !b.draft() {
		noteDir = path.Dir(b.id)
		if noteDir == "." {
			noteDir = ""
		}
	}
	link, err := m.store.AttachFile(noteDir, src)
	if err != nil {
		m.setError(i18n.T(i18n.AppAttachFailed, err.Error()))
		return ""
	}
	if kind == "cover" {
		m.content.SetFrontMatter("cover", link)
		m.preview.SetContent(m.content.Value())
		m.refreshModified()
		m.setStatus(i18n.T(i18n.AppCoverSet, link))
		return link
	}
	name := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	target := link
	if strings.ContainsAny(target, " ()") {
		target = "<" + target + ">"
	}
	var md string
	switch kind {
	case "image":
		md = "![" + name + "](" + target + ")"
	case "video":
		md = "[▶ " + name + "](" + target + ")"
	case "audio":
		md = "[♪ " + name + "](" + target + ")"
	default:
		md = "[📎 " + filepath.Base(src) + "](" + target + ")"
	}
	if ownLine {
		m.content.InsertBlock(md)
	} else {
		m.content.InsertText(md)
	}
	m.preview.SetContent(m.content.Value())
	m.refreshModified()
	if b.draft() {
		return i18n.T(i18n.AppSavedWithFolder, link)
	}
	return link
}
