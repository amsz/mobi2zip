package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// fileState is where a file of a group stands.
type fileState int

const (
	fileWaiting fileState = iota
	fileRunning
	fileDone
	fileFailed
	fileSkipped
)

// fileItem is one document of a group: a .mobi, .epub or .pdf.
type fileItem struct {
	name   string // the file name, without its directory
	path   string // the full path
	format string // "MOBI", "EPUB" or "PDF"
	size   int64
	state  fileState
	pct    int    // how far the conversion got, 0-100
	dir    string // the folder under the group's root, for files nested
	zip    string // the zip written, once done
	moved  string // where the original went, once archived
	err    string // why it failed, or why it was skipped
}

// group is what one dropped directory holds.
type group struct {
	id    int
	root  string // the directory as dropped, shown to the user
	files []*fileItem
	open  bool // its files show under it
}

// done and failed count the files that ended, and failed.
func (g *group) done() int {
	n := 0
	for _, f := range g.files {
		if f.state == fileDone || f.state == fileFailed || f.state == fileSkipped {
			n++
		}
	}
	return n
}

func (g *group) failed() int {
	n := 0
	for _, f := range g.files {
		if f.state == fileFailed {
			n++
		}
	}
	return n
}

// running reports whether a file of the group is being converted.
func (g *group) running() bool {
	for _, f := range g.files {
		if f.state == fileRunning {
			return true
		}
	}
	return false
}

// finished reports whether every file of the group ended.
func (g *group) finished() bool {
	return len(g.files) > 0 && g.done() == len(g.files)
}

// progress is how much of the group is converted, from 0 to 1.
func (g *group) progress() float64 {
	if len(g.files) == 0 {
		return 0
	}
	sum := 0.0
	for _, f := range g.files {
		switch f.state {
		case fileDone, fileFailed, fileSkipped:
			sum++
		case fileRunning:
			sum += float64(f.pct) / 100
		}
	}
	return sum / float64(len(g.files))
}

// formats the app converts, in the order a group's files are listed.
var formats = []string{".mobi", ".epub", ".pdf"}

// scan collects the documents under the paths dropped, one group per
// directory, and one per file dropped on its own.
func (a *app) scan(paths []string) {
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		g := &group{root: p, open: true}
		if st.IsDir() {
			g.files = walkDir(p)
		} else if isDocument(p) {
			g.files = []*fileItem{item(p, st.Size())}
			g.root = filepath.Dir(p)
		} else {
			continue
		}
		if len(g.files) == 0 {
			continue
		}
		a.nextID++
		g.id = a.nextID
		a.groups = append(a.groups, g)
	}
	a.redraw()
}

// walkDir lists the documents under root, sorted by path.
func walkDir(root string) []*fileItem {
	var items []*fileItem
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			// The archive folder the app fills is never scanned again.
			if d != nil && d.IsDir() && p != root && d.Name() == archiveDirName {
				return filepath.SkipDir
			}
			return nil
		}
		if !isDocument(p) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		it := item(p, info.Size())
		if rel, err := filepath.Rel(root, filepath.Dir(p)); err == nil && rel != "." {
			it.dir = rel
		}
		items = append(items, it)
		return nil
	})
	sort.SliceStable(items, func(i, j int) bool { return items[i].path < items[j].path })
	return items
}

// item is a file of a group.
func item(p string, size int64) *fileItem {
	return &fileItem{
		name:   filepath.Base(p),
		path:   p,
		format: strings.ToUpper(strings.TrimPrefix(filepath.Ext(p), ".")),
		size:   size,
	}
}

// isDocument reports whether p is a format the app converts.
func isDocument(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	for _, f := range formats {
		if ext == f {
			return true
		}
	}
	return false
}

// archiveDirName is the default folder the converted originals move to.
const archiveDirName = "_bak"
