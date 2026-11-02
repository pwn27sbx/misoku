// Package kitty draws pictures with the Kitty graphics protocol: it encodes
// the escape sequences, decides whether the terminal can show them, and
// serializes them with the rest of the program's output. Half blocks stay the
// fallback everywhere else (see internal/preview).
package kitty

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// Mode says how the preview draws a picture.
type Mode int

const (
	// Blocks is the half-block drawing: two pixels per cell, works anywhere.
	Blocks Mode = iota
	// Graphics sends the image to the terminal (Kitty graphics protocol).
	Graphics
)

// chunkSize is how many base64 bytes one escape sequence carries. The protocol
// requires each chunk to stay under 4096 bytes.
const chunkSize = 4096

// placeholder is the Private Use rune the terminal replaces with the image.
const placeholder = '\U0010EEEE'

// maxID is the largest image id a placeholder can name: the id travels as a
// 256-colour foreground (38;5;N), whose palette holds 0…255.
const maxID = 255

// Detect chooses how pictures are drawn, from the environment only: a query to
// the terminal would arrive at Bubble Tea's input loop as typed runes, and a
// reply cannot be told apart from a paste.
//
// MISOKU_IMAGES wins over everything. TMUX and screen never pass graphics
// through, so they get half blocks. herdr renders graphics itself since 0.9, so
// a herdr pane counts as capable even though it masks the outer terminal.
func Detect(env func(string) string, isTTY bool) Mode {
	if env == nil {
		return Blocks
	}
	switch strings.ToLower(strings.TrimSpace(env("MISOKU_IMAGES"))) {
	case "blocks", "half", "off":
		return Blocks
	case "kitty", "graphics", "on":
		return Graphics
	}
	if !isTTY {
		return Blocks
	}
	term := env("TERM")
	herdr := env("HERDR_ENV") == "1"
	if env("TMUX") != "" || strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux") {
		return Blocks
	}
	if !herdr && (env("ZELLIJ") != "" || env("CMUX_SOCKET") != "" || env("CMUX_PANE_ID") != "") {
		return Blocks
	}
	if herdr {
		return Graphics
	}
	switch strings.ToLower(env("TERM_PROGRAM")) {
	case "ghostty", "kitty", "wezterm":
		return Graphics
	}
	if strings.Contains(term, "kitty") || strings.Contains(term, "ghostty") {
		return Graphics
	}
	if env("KITTY_WINDOW_ID") != "" || env("KITTY_PID") != "" {
		return Graphics
	}
	return Blocks
}

// Sender is the one output the program has: frames and graphics go through it so
// a chunked transmit is never cut in half by a frame.
type Sender interface {
	Send(seqs ...string) error
}

// Transmit returns the escape sequences that send a PNG under id and create its
// virtual placement (U=1) for a cols×rows cell box. The image is displayed by
// the placeholders Placeholders returns, not by this sequence.
//
// q=2 keeps the terminal quiet: a reply would land in Bubble Tea's input.
func Transmit(id, cols, rows int, png []byte) []string {
	if id <= 0 || id > maxID || len(png) == 0 || cols <= 0 || rows <= 0 {
		return nil
	}
	data := base64.StdEncoding.EncodeToString(png)
	head := fmt.Sprintf("a=T,f=100,t=d,i=%d,U=1,c=%d,r=%d,q=2", id, cols, rows)
	var seqs []string
	for first := true; data != "" || first; first = false {
		n := min(chunkSize, len(data))
		chunk := data[:n]
		data = data[n:]
		var params string
		switch {
		case first:
			params = head
			if data != "" {
				params += ",m=1"
			}
		case data != "":
			params = "m=1"
		default:
			params = "m=0"
		}
		seqs = append(seqs, "\x1b_G"+params+";"+chunk+"\x1b\\")
	}
	return seqs
}

// Delete returns the sequence that removes image id and frees its data. It
// never touches other programs' images: d=A would.
func Delete(id int) string {
	return fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id)
}

// Placeholders returns the cells that display image id in a cols×rows box: one
// line per row, each cell U+10EEEE followed by its row and column diacritics and
// painted in the image id as a 256-colour foreground. The closing reset keeps
// the pane's padding out of that colour.
func Placeholders(id, cols, rows int) []string {
	if id <= 0 || id > maxID || cols <= 0 || rows <= 0 {
		return nil
	}
	lines := make([]string, rows)
	for y := 0; y < rows; y++ {
		var b strings.Builder
		fmt.Fprintf(&b, "\x1b[38;5;%dm", id)
		for x := 0; x < cols; x++ {
			b.WriteRune(placeholder)
			b.WriteRune(diacritic(y))
			b.WriteRune(diacritic(x))
		}
		b.WriteString("\x1b[39m")
		lines[y] = b.String()
	}
	return lines
}

// diacritic maps a row or column index to its combining mark. Past the end of
// the table the cell inherits the row and column of the cell to its left, which
// is what the protocol specifies for a missing diacritic.
func diacritic(n int) rune {
	if n < 0 || n >= len(rowColumn) {
		return 0
	}
	return rowColumn[n]
}

// rowColumn is kitty's rowcolumn-diacritics.txt, in index order. Every rune is a
// combining mark of class 230 that Unicode never fuses into a composed letter.
var rowColumn = []rune{
	0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F, 0x0346, 0x034A, 0x034B, 0x034C,
	0x0350, 0x0351, 0x0352, 0x0357, 0x035B, 0x0363, 0x0364, 0x0365, 0x0366, 0x0367, 0x0368, 0x0369,
	0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F, 0x0483, 0x0484, 0x0485, 0x0486, 0x0487, 0x0592,
	0x0593, 0x0594, 0x0595, 0x0597, 0x0598, 0x0599, 0x059C, 0x059D, 0x059E, 0x059F, 0x05A0, 0x05A1,
	0x05A8, 0x05A9, 0x05AB, 0x05AC, 0x05AF, 0x05C4, 0x0610, 0x0611, 0x0612, 0x0613, 0x0614, 0x0615,
	0x0616, 0x0617, 0x0657, 0x0658, 0x0659, 0x065A, 0x065B, 0x065D, 0x065E, 0x06D6, 0x06D7, 0x06D8,
	0x06D9, 0x06DA, 0x06DB, 0x06DC, 0x06DF, 0x06E0, 0x06E1, 0x06E2, 0x06E4, 0x06E7, 0x06E8, 0x06EB,
	0x06EC, 0x0730, 0x0732, 0x0733, 0x0735, 0x0736, 0x073A, 0x073D, 0x073F, 0x0740, 0x0741, 0x0743,
	0x0745, 0x0747, 0x0749, 0x074A, 0x07EB, 0x07EC, 0x07ED, 0x07EE, 0x07EF, 0x07F0, 0x07F1, 0x07F3,
	0x0816, 0x0817, 0x0818, 0x0819, 0x081B, 0x081C, 0x081D, 0x081E, 0x081F, 0x0820, 0x0821, 0x0822,
	0x0823, 0x0825, 0x0826, 0x0827, 0x0829, 0x082A, 0x082B, 0x082C, 0x082D, 0x0951, 0x0953, 0x0954,
	0x0F82, 0x0F83, 0x0F86, 0x0F87, 0x135D, 0x135E, 0x135F, 0x17DD, 0x193A, 0x1A17, 0x1A75, 0x1A76,
	0x1A77, 0x1A78, 0x1A79, 0x1A7A, 0x1A7B, 0x1A7C, 0x1B6B, 0x1B6D, 0x1B6E, 0x1B6F, 0x1B70, 0x1B71,
	0x1B72, 0x1B73, 0x1CD0, 0x1CD1, 0x1CD2, 0x1CDA, 0x1CDB, 0x1CE0, 0x1DC0, 0x1DC1, 0x1DC3, 0x1DC4,
	0x1DC5, 0x1DC6, 0x1DC7, 0x1DC8, 0x1DC9, 0x1DCB, 0x1DCC, 0x1DD1, 0x1DD2, 0x1DD3, 0x1DD4, 0x1DD5,
	0x1DD6, 0x1DD7, 0x1DD8, 0x1DD9, 0x1DDA, 0x1DDB, 0x1DDC, 0x1DDD, 0x1DDE, 0x1DDF, 0x1DE0, 0x1DE1,
	0x1DE2, 0x1DE3, 0x1DE4, 0x1DE5, 0x1DE6, 0x1DFE, 0x20D0, 0x20D1, 0x20D4, 0x20D5, 0x20D6, 0x20D7,
	0x20DB, 0x20DC, 0x20E1, 0x20E7, 0x20E9, 0x20F0, 0x2CEF, 0x2CF0, 0x2CF1, 0x2DE0, 0x2DE1, 0x2DE2,
	0x2DE3, 0x2DE4, 0x2DE5, 0x2DE6, 0x2DE7, 0x2DE8, 0x2DE9, 0x2DEA, 0x2DEB, 0x2DEC, 0x2DED, 0x2DEE,
	0x2DEF, 0x2DF0, 0x2DF1, 0x2DF2, 0x2DF3, 0x2DF4, 0x2DF5, 0x2DF6, 0x2DF7, 0x2DF8, 0x2DF9, 0x2DFA,
	0x2DFB, 0x2DFC, 0x2DFD, 0x2DFE, 0x2DFF, 0xA66F, 0xA67C, 0xA67D, 0xA6F0, 0xA6F1, 0xA8E0, 0xA8E1,
	0xA8E2, 0xA8E3, 0xA8E4, 0xA8E5, 0xA8E6, 0xA8E7, 0xA8E8, 0xA8E9, 0xA8EA, 0xA8EB, 0xA8EC, 0xA8ED,
	0xA8EE, 0xA8EF, 0xA8F0, 0xA8F1, 0xAAB0, 0xAAB2, 0xAAB3, 0xAAB7, 0xAAB8, 0xAABE, 0xAABF, 0xAAC1,
	0xFE20, 0xFE21, 0xFE22, 0xFE23, 0xFE24, 0xFE25, 0xFE26, 0x10A0F, 0x10A38, 0x1D185, 0x1D186, 0x1D187,
	0x1D188, 0x1D189, 0x1D1AA, 0x1D1AB, 0x1D1AC, 0x1D1AD, 0x1D242, 0x1D243, 0x1D244,
}
