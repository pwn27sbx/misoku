package tutor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/export"
	"github.com/pwn27sbx/misoku/internal/tutor"
)

// exportStepCheck runs one step's disk checks against the installed tutorial,
// with the PDFs that exist in dir at that moment.
func exportStepCheck(t *testing.T, l tutor.Lesson, step int, dir string) bool {
	t.Helper()
	store := &tutor.Store{}
	store.RecheckDisk(l, tutor.DirEnv(dir))
	return store.Passed(l, step)
}

// exportPDF writes the PDF the app writes: same target, same call.
func exportPDF(t *testing.T, target, title string) {
	t.Helper()
	if err := export.WriteFile("# "+title+"\n", export.Options{Title: title}, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("the export left no PDF at %s: %v", target, err)
	}
}

// S5: A13's three steps pass once the PDF is really exported where the lesson
// says: next to the note with ":export", with the given name with
// ":export informe.pdf", and next to A00 with "space P".
func TestExportStepsPassWithRealPDFs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tutor")
	if err := tutor.Install(tutor.Bundled(), dir, false); err != nil {
		t.Fatal(err)
	}
	cat := tutor.Default()
	l, ok := cat.Lesson("3 Avanzado/A13 Exportar a PDF.md")
	if !ok {
		t.Fatal("A13 is not in the catalog")
	}
	if len(l.Steps) != 3 {
		t.Fatalf("A13 has %d steps, want 3", len(l.Steps))
	}
	noteDir := filepath.Join(dir, "3 Avanzado")

	// ":export" without an argument exports the open note next to it.
	if exportStepCheck(t, l, 1, dir) {
		t.Fatal("step 1 passed before anything was exported")
	}
	exportPDF(t, export.Target("", noteDir, "A13 Exportar a PDF"), "A13 Exportar a PDF")
	if !exportStepCheck(t, l, 1, dir) {
		t.Fatal("step 1 did not pass with the exported PDF next to the note")
	}

	// ":export informe.pdf" writes the named file in the note's folder.
	if exportStepCheck(t, l, 2, dir) {
		t.Fatal("step 2 passed before the named export")
	}
	exportPDF(t, export.Target("informe.pdf", noteDir, "A13 Exportar a PDF"), "informe.pdf")
	if !exportStepCheck(t, l, 2, dir) {
		t.Fatal("step 2 did not pass with informe.pdf in the note's folder")
	}

	// "space P" on A00 writes A00's PDF in A00's folder.
	if exportStepCheck(t, l, 3, dir) {
		t.Fatal("step 3 passed before A00 was exported")
	}
	exportPDF(t, export.Target("", noteDir, "A00 Avanzado"), "A00 Avanzado")
	if !exportStepCheck(t, l, 3, dir) {
		t.Fatal("step 3 did not pass with A00 Avanzado.pdf in the route folder")
	}
}
