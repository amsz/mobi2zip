package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// archiveOf returns where g's file f goes under the archive, keeping the
// path it has below the dropped directory.
func archivePath(archive string, g *group, f *fileItem) string {
	rel := f.name
	if g != nil {
		if r, err := filepath.Rel(g.root, f.path); err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			rel = r
		}
	}
	return filepath.Join(archive, rel)
}

// moveToArchive moves a converted original into the archive folder. It
// returns where the file went, or "" when no archive is set.
func (a *app) moveToArchive(g *group, f *fileItem) (string, error) {
	if a.archive == "" {
		return "", nil
	}
	dest := freePath(archivePath(a.archive, g, f))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(f.path, dest); err != nil {
		// Another volume: copy, then remove the original.
		if err := copyFile(f.path, dest); err != nil {
			return "", err
		}
		if err := os.Remove(f.path); err != nil {
			return "", err
		}
	}
	return dest, nil
}

// freePath returns p, or "name (1).ext" beside it while that is taken.
func freePath(p string) string {
	if _, err := os.Stat(p); err != nil {
		return p
	}
	ext := filepath.Ext(p)
	stem := strings.TrimSuffix(p, ext)
	for i := 1; ; i++ {
		cand := fmt.Sprintf("%s (%d)%s", stem, i, ext)
		if _, err := os.Stat(cand); err != nil {
			return cand
		}
	}
}

// copyFile copies src to dst, keeping its permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, st.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}
