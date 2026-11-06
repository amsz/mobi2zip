package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// devSeed fills the list at start, for looking at the views without dropping
// anything: MOBI2ZIP_DEMO names a directory, MOBI2ZIP_DEMO_FAKE marks its
// files ended, to check the rows, and MOBI2ZIP_DEMO_AUTOSTART converts them.
func (a *app) devSeed() {
	dir := os.Getenv("MOBI2ZIP_DEMO")
	if dir == "" {
		return
	}
	a.scan([]string{dir})

	if os.Getenv("MOBI2ZIP_DEMO_FAKE") != "" {
		for _, g := range a.groups {
			for i, f := range g.files {
				switch i % 4 {
				case 0:
					f.state = fileDone
					f.zip = strings.TrimSuffix(f.name, filepath.Ext(f.name)) + ".zip"
				case 1:
					f.state, f.err = fileFailed, "无法读取：文件已加密"
				case 2:
					f.state, f.err = fileSkipped, "zip 已存在，跳过"
				}
			}
		}
	}
	if os.Getenv("MOBI2ZIP_DEMO_AUTOSTART") != "" {
		a.start()
	}
	fmt.Fprintf(os.Stderr, "demo %q: %d groups\n", dir, len(a.groups))
}
