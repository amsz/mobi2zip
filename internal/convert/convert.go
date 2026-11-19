// Package convert turns a .mobi, .epub or .pdf into a zip of its pictures.
package convert

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Progress reports how far a conversion got, from 0 to 100, and what it is
// doing.
type Progress func(pct int, note string)

// Converter turns documents into zips.
type Converter struct {
	// Calibre is the path of calibre's ebook-convert, "" when none was
	// found: .epub files convert without it, the others need it.
	Calibre string
}

// Convert writes the pictures of src to dst as a zip, in page order.
func (c *Converter) Convert(ctx context.Context, src, dst string, progress Progress) (int, error) {
	if progress == nil {
		progress = func(int, string) {}
	}

	var files []*zip.File
	var reader *zip.ReadCloser
	var err error
	switch strings.ToLower(filepath.Ext(src)) {
	case ".epub":
		progress(5, "读取 EPUB")
		reader, files, err = openEPUB(src)
	default:
		if c.Calibre == "" {
			return 0, fmt.Errorf("找不到 calibre 的 ebook-convert，%s 需要它", strings.ToUpper(strings.TrimPrefix(filepath.Ext(src), ".")))
		}
		var epub string
		epub, err = c.toEPUB(ctx, src, progress)
		if err != nil {
			return 0, err
		}
		defer os.Remove(epub)
		progress(70, "读取图片")
		reader, files, err = openEPUB(epub)
	}
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	if len(files) == 0 {
		return 0, fmt.Errorf("没有找到图片")
	}
	progress(80, "打包")
	n, err := writeZip(dst, files)
	if err != nil {
		return 0, err
	}
	progress(100, "完成")
	return n, nil
}

// writeZip writes the pictures as 001.jpg, 002.jpg… storing them as they
// are: JPEG and PNG are compressed already.
func writeZip(dst string, files []*zip.File) (int, error) {
	width := len(strconv.Itoa(len(files)))
	if width < 3 {
		width = 3
	}
	out, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	for i, f := range files {
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext == ".jpeg" {
			ext = ".jpg"
		}
		name := fmt.Sprintf("%0*d%s", width, i+1, ext)
		h := &zip.FileHeader{Name: name, Method: zip.Store}
		h.Modified = f.Modified
		w, err := zw.CreateHeader(h)
		if err != nil {
			return 0, err
		}
		r, err := f.Open()
		if err != nil {
			return 0, err
		}
		if _, err := io.Copy(w, r); err != nil {
			r.Close()
			return 0, err
		}
		r.Close()
	}
	if err := zw.Close(); err != nil {
		return 0, err
	}
	return len(files), nil
}

// toEPUB has calibre convert src to an EPUB in a temporary directory, and
// returns its path.
func (c *Converter) toEPUB(ctx context.Context, src string, progress Progress) (string, error) {
	dir, err := os.MkdirTemp("", "mobi2zip-")
	if err != nil {
		return "", err
	}
	dst := filepath.Join(dir, "book.epub")

	args := []string{src, dst, "--no-default-epub-cover", "--disable-font-rescaling"}
	cmd := exec.CommandContext(ctx, c.Calibre, args...)
	hideWindow(cmd)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	progress(10, "calibre 转换中")

	if err := cmd.Start(); err != nil {
		os.RemoveAll(dir)
		return "", err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// calibre prints the percentage of its work as it goes, as " 42%".
	pct := regexp.MustCompile(`(\d{1,3})%`)
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			<-done
			os.RemoveAll(dir)
			return "", ctx.Err()
		case err := <-done:
			if err != nil {
				os.RemoveAll(dir)
				return "", fmt.Errorf("calibre 失败：%v\n%s", err, tail(out.String(), 4))
			}
			return dst, nil
		case <-tick.C:
			if m := pct.FindAllStringSubmatch(out.String(), -1); len(m) > 0 {
				if v, err := strconv.Atoi(m[len(m)-1][1]); err == nil {
					// calibre's own 0-100 maps onto the first 60% of the job.
					progress(10+v*3/5, "calibre 转换中")
				}
			}
		}
	}
}

// CalibreVersion asks calibre for its version, to tell a working install
// from a path that only exists.
func CalibreVersion(ctx context.Context, exe string) (string, error) {
	if exe == "" {
		return "", errNoCalibre
	}
	cmd := exec.CommandContext(ctx, exe, "--version")
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v：%s", err, tail(string(out), 2))
	}
	line := strings.TrimSpace(strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0])
	if line == "" {
		return "", errNoCalibre
	}
	return line, nil
}

// errNoCalibre is returned when calibre cannot be asked for its version.
var errNoCalibre = errors.New("没有可用的 calibre")

// tail returns the last n non-empty lines of s.
func tail(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	var kept []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			kept = append(kept, strings.TrimSpace(l))
		}
	}
	if len(kept) > n {
		kept = kept[len(kept)-n:]
	}
	return strings.Join(kept, "\n")
}

// FindCalibre returns the path of calibre's ebook-convert, or "".
func FindCalibre() string {
	if p := os.Getenv("CALIBRE_PATH"); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
		if st, err := os.Stat(filepath.Join(p, "ebook-convert.exe")); err == nil && !st.IsDir() {
			return filepath.Join(p, "ebook-convert.exe")
		}
	}
	if p, err := exec.LookPath("ebook-convert"); err == nil {
		return p
	}
	candidates := []string{
		"/Applications/calibre.app/Contents/MacOS/ebook-convert",
		"/usr/bin/ebook-convert",
		"/usr/local/bin/ebook-convert",
		"/opt/calibre/ebook-convert",
		filepath.Join(os.Getenv("ProgramFiles"), "Calibre2", "ebook-convert.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Calibre2", "ebook-convert.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "calibre", "ebook-convert.exe"),
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// errNoPackage is returned when an EPUB does not say where its content is.
var errNoPackage = errors.New("EPUB 里没有找到 OPF（可能不是有效的 EPUB）")
