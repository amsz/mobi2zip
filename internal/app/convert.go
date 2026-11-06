package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/egoist/mygo"
)

// workers is how many documents convert at once: calibre is heavy, so a
// couple at a time.
const workers = 2

// job is one document of a group to convert.
type job struct {
	g *group
	f *fileItem
}

// start converts the files that wait, a couple at a time, and posts each
// change to the main thread.
func (a *app) start() {
	if a.busy || a.conv == nil {
		return
	}
	var waiting []job
	for _, g := range a.groups {
		for _, f := range g.files {
			if f.state == fileWaiting {
				waiting = append(waiting, job{g: g, f: f})
			}
		}
	}
	if len(waiting) == 0 {
		return
	}
	a.busy = true

	go func() {
		jobs := make(chan job)
		var wg sync.WaitGroup
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobs {
					a.convertOne(j)
				}
			}()
		}
		for _, j := range waiting {
			jobs <- j
		}
		close(jobs)
		wg.Wait()
		a.finish()
	}()
}

// convertOne turns one document into the zip beside it, and moves the
// original into the archive.
func (a *app) convertOne(j job) {
	f := j.f
	dst := strings.TrimSuffix(f.path, filepath.Ext(f.path)) + ".zip"
	if _, err := os.Stat(dst); err == nil {
		a.post(func() {
			f.state = fileSkipped
			f.err = "zip 已存在，跳过"
		})
		return
	}

	a.post(func() {
		f.state = fileRunning
		f.pct = 0
	})

	_, err := a.conv.Convert(context.Background(), f.path, dst, func(pct int, note string) {
		a.post(func() {
			if pct > f.pct {
				f.pct = pct
			}
		})
	})
	if err != nil {
		msg := err.Error()
		if i := strings.IndexByte(msg, '\n'); i > 0 {
			msg = msg[:i]
		}
		a.post(func() {
			f.state = fileFailed
			f.err = msg
		})
		return
	}

	a.post(func() {
		f.state = fileDone
		f.pct = 100
		f.zip = filepath.Base(dst)
	})
	if a.archive != "" {
		if dest, err := a.moveToArchive(j.g, f); err == nil {
			a.post(func() { f.moved = dest })
		} else {
			a.post(func() { f.err = "已转换，但移到归档失败：" + err.Error() })
		}
	}
}

// finish ends a run: it clears the flag, tells the user, and posts a
// notification when the system shows them.
func (a *app) finish() {
	a.post(func() {
		a.busy = false
		done, skipped, failed := a.totals()
		a.toast = fmt.Sprintf("完成：成功 %d · 跳过 %d · 失败 %d", done, skipped, failed)
		if done > 0 && a.settings.Notify && mygo.NotificationsSupported() {
			body := fmt.Sprintf("成功 %d · 跳过 %d · 失败 %d", done, skipped, failed)
			if a.archive != "" {
				body += "，已归档到 " + a.archive
			}
			mygo.NewNotification(mygo.NotificationOptions{Title: "漫画解包", Body: body}).Show()
		}
	})
}

// totals counts how the files of every group ended.
func (a *app) totals() (done, skipped, failed int) {
	for _, g := range a.groups {
		for _, f := range g.files {
			switch f.state {
			case fileDone:
				done++
			case fileSkipped:
				skipped++
			case fileFailed:
				failed++
			}
		}
	}
	return done, skipped, failed
}
