package app

import (
	"os"
	"path/filepath"
	"testing"
)

// dropping a directory lists its documents, skipping other files and the
// archive folder.
func TestScanDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "卷 1")
	if err := os.MkdirAll(filepath.Join(sub, archiveDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.mobi", "b.pdf", "c.txt"} {
		if err := os.WriteFile(filepath.Join(sub, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(sub, archiveDirName, "old.mobi"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	a := &app{}
	a.scan([]string{dir})

	if len(a.groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(a.groups))
	}
	g := a.groups[0]
	if len(g.files) != 2 {
		t.Fatalf("files = %d, want 2", len(g.files))
	}
	if g.files[0].format != "MOBI" || g.files[1].format != "PDF" {
		t.Fatalf("formats = %q, %q", g.files[0].format, g.files[1].format)
	}
	if !g.open {
		t.Fatal("a new group should show its files")
	}
}

// dropping a single document makes a group of it.
func TestScanFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "book.epub")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	a := &app{}
	a.scan([]string{p})

	if len(a.groups) != 1 || len(a.groups[0].files) != 1 {
		t.Fatalf("groups = %d", len(a.groups))
	}
	if got := a.groups[0].files[0].format; got != "EPUB" {
		t.Fatalf("format = %q", got)
	}
	if a.groups[0].root != dir {
		t.Fatalf("root = %q, want %q", a.groups[0].root, dir)
	}
}

// progress counts finished files whole and the running one by its share.
func TestGroupProgress(t *testing.T) {
	g := &group{files: []*fileItem{
		{state: fileDone},
		{state: fileRunning, pct: 50},
		{state: fileWaiting},
		{state: fileWaiting},
	}}
	if got := g.progress(); got != 0.375 {
		t.Fatalf("progress = %v, want 0.375", got)
	}
	if g.finished() {
		t.Fatal("a group with files waiting is not finished")
	}
}
