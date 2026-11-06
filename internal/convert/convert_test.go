package convert

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// createEPUB writes a small EPUB whose pages reference its pictures out of
// name order, to check that the spine decides the order.
func createEPUB(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "book.epub")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	write := func(name, body string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}

	write("mimetype", "application/epub+zip")
	write("META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`)
	write("OEBPS/content.opf", `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
<manifest>
<item id="a" href="img/a.jpg" media-type="image/jpeg"/>
<item id="b" href="img/b.jpeg" media-type="image/jpeg"/>
<item id="c" href="img/c.jpg" media-type="image/jpeg"/>
<item id="u" href="img/unused.jpg" media-type="image/jpeg"/>
<item id="p1" href="text/1.xhtml" media-type="application/xhtml+xml"/>
<item id="p2" href="text/2.xhtml" media-type="application/xhtml+xml"/>
<item id="p3" href="text/3.xhtml" media-type="application/xhtml+xml"/>
</manifest>
<spine><itemref idref="p1"/><itemref idref="p2"/><itemref idref="p3"/></spine>
</package>`)
	write("OEBPS/text/1.xhtml", `<html><body><img src="../img/c.jpg"/></body></html>`)
	write("OEBPS/text/2.xhtml", `<html><body><image xlink:href="../img/a.jpg"/></body></html>`)
	write("OEBPS/text/3.xhtml", `<html><body><img src="../img/b.jpeg"/></body></html>`)
	for name, body := range map[string]string{
		"OEBPS/img/a.jpg": "A", "OEBPS/img/b.jpeg": "B",
		"OEBPS/img/c.jpg": "C", "OEBPS/img/unused.jpg": "U",
	} {
		write(name, body)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

// the spine decides the order, and pictures it skips come last.
func TestEPUBPictures(t *testing.T) {
	src := createEPUB(t, t.TempDir())
	r, files, err := openEPUB(src)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	var names []string
	for _, f := range files {
		names = append(names, filepath.Base(f.Name))
	}
	want := []string{"c.jpg", "a.jpg", "b.jpeg", "unused.jpg"}
	if len(names) != len(want) {
		t.Fatalf("pictures = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("pictures = %v, want %v", names, want)
		}
	}
}

// an EPUB converts to a zip numbered in page order.
func TestConvertEPUB(t *testing.T) {
	dir := t.TempDir()
	src := createEPUB(t, dir)
	dst := filepath.Join(dir, "out.zip")

	c := &Converter{}
	n, err := c.Convert(context.Background(), src, dst, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("images = %d, want 4", n)
	}
	zr, err := zip.OpenReader(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	want := map[string]string{"001.jpg": "C", "002.jpg": "A", "003.jpg": "B", "004.jpg": "U"}
	if len(zr.File) != len(want) {
		t.Fatalf("entries = %d, want %d", len(zr.File), len(want))
	}
	for _, f := range zr.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 8)
		n, _ := r.Read(buf)
		r.Close()
		if got := string(buf[:n]); got != want[f.Name] {
			t.Fatalf("%s = %q, want %q", f.Name, got, want[f.Name])
		}
		if f.Method != zip.Store {
			t.Fatalf("%s is compressed", f.Name)
		}
	}
}

// a real MOBI converts through calibre, when a sample and calibre are at
// hand: set MOBI2ZIP_SAMPLE to the file to check it.
func TestConvertMOBISample(t *testing.T) {
	sample := os.Getenv("MOBI2ZIP_SAMPLE")
	if sample == "" {
		t.Skip("set MOBI2ZIP_SAMPLE to a .mobi file to run this")
	}
	c := &Converter{Calibre: FindCalibre()}
	if c.Calibre == "" {
		t.Skip("calibre is not installed")
	}
	dst := filepath.Join(t.TempDir(), "out.zip")
	n, err := c.Convert(context.Background(), sample, dst, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("images: %d", n)
	if n < 100 {
		t.Fatalf("images = %d, want the book's pages", n)
	}
	zr, err := zip.OpenReader(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	if len(zr.File) != n {
		t.Fatalf("entries = %d, images = %d", len(zr.File), n)
	}
}
