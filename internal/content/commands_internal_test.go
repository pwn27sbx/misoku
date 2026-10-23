package content

import (
	"slices"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

// A command the catalog leaves out is rejected even if parseVimCommand
// still has a case for it, so the palette can never miss one.
func TestACommandMissingFromTheCatalogIsRejected(t *testing.T) {
	saved := Commands
	t.Cleanup(func() { Commands = saved })
	Commands = slices.DeleteFunc(slices.Clone(saved), func(c Command) bool { return slices.Contains(c.Names, "graph") })
	msg, ok := parseVimCommand("graph", "").(core.StatusMsg)
	if !ok || !strings.HasPrefix(msg.Text, "E492") {
		t.Fatalf(":graph without a catalog entry must be rejected, got %v", msg)
	}
}
