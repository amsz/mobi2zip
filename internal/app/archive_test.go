package app

import (
	"os"
	"path/filepath"
	"testing"
)

// a converted original goes into the archive under the path it had below
// the directory dropped.
func TestMoveToArchive(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "卷 1")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(sub, "a.mobi")
	if err := os.WriteFile(src, []byte("book"), 0o644); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(t.TempDir(), "归档")
	a := &app{archive: archive}
	g := &group{root: root}
	f := &fileItem{name: "a.mobi", path: src}

	dest, err := a.moveToArchive(g, f)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(archive, "卷 1", "a.mobi")
	if dest != want {
		t.Fatalf("moved to %q, want %q", dest, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("archived file: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("the original should be gone")
	}
}

// a name already in the archive gets a number rather than replacing it.
func TestMoveToArchiveKeepsBoth(t *testing.T) {
	root := t.TempDir()
	archive := t.TempDir()
	src := filepath.Join(root, "a.mobi")

	a := &app{archive: archive}
	g := &group{root: root}
	for i := 0; i < 2; i++ {
		if err := os.WriteFile(src, []byte{byte('a' + i)}, 0o644); err != nil {
			t.Fatal(err)
		}
		f := &fileItem{name: "a.mobi", path: src}
		if _, err := a.moveToArchive(g, f); err != nil {
			t.Fatal(err)
		}
	}

	first, err := os.ReadFile(filepath.Join(archive, "a.mobi"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(archive, "a (1).mobi"))
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != "a" || string(second) != "b" {
		t.Fatalf("archived %q and %q", first, second)
	}
}
