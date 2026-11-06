package app

import (
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// listRow is one row of the group list: a group's header, or one of its
// files under it.
type listRow struct {
	g *group
	f *fileItem
}

// view builds the window from the app's state, every frame.
func (a *app) view(c *ui.Context) {
	t := *c.Theme()
	a.pal = paletteFor(&t)
	p := a.pal
	t.Accent, t.AccentText = p.accent, p.onAccent
	c.SetTheme(&t)
	c.Root().Fill().Background(p.bg)

	ui.Column(c).Fill().Children(func() {
		a.header(c)
		a.dropZone(c)
		a.listView(c)
		a.footer(c)
	})

	if a.toast != "" {
		c.Toast(a.toast)
		a.toast = ""
	}
}

// titleBarHeight is how tall the strip along the top is: the window's
// buttons fill it on Windows and Linux.
const titleBarHeight = 52

// header is the title of the window. It draws the title bar the hidden one
// leaves to the view, and dragging it moves the window.
func (a *app) header(c *ui.Context) {
	p := a.pal
	bar := c.TitleBar()
	ui.Row(c).Height(max(bar.Height, titleBarHeight)).
		Padding(0, bar.Right+22, 0, bar.Left+22).Gap(10).AlignItems(ui.Center).DragWindow().
		Children(func() {
			ui.Column(c).Grow(1).Gap(1).Children(func() {
				ui.Text(c, "漫画解包").FontSize(18).FontWeight(700).TextColor(p.text)
				ui.Text(c, "把文件夹拖进来，转成同名的 zip").FontSize(12).TextColor(p.muted)
			})
		})
}

// dropZone is where directories are dropped, and what says it would take
// them while they are over it.
func (a *app) dropZone(c *ui.Context) {
	p := a.pal
	zone := ui.Column(c).Height(84).Margin(6, 22, 10, 22).Radius(14).
		Border(1, p.hairline).Background(p.panel).Center().Gap(6).Bind(&a.zone)
	over := zone.FileDragOver()
	if over {
		zone.Border(1, p.accent)
	}
	zone.Children(func() {
		ui.Icon(c, iconFolder).Size(22, 22).TextColor(p.faint)
		ui.Text(c, "把文件夹或 .mobi / .epub / .pdf 拖到这里").FontSize(13).TextColor(p.muted)
	})
	if files := zone.DroppedFiles(); len(files) > 0 {
		a.scan(files)
	}
}

// listView lists the groups, and the files of those opened.
func (a *app) listView(c *ui.Context) {
	p := a.pal
	rows := a.rows[:0]
	for _, g := range a.groups {
		rows = append(rows, listRow{g: g})
		if g.open {
			for _, f := range g.files {
				rows = append(rows, listRow{g: g, f: f})
			}
		}
	}
	a.rows = rows

	if len(rows) == 0 {
		ui.Column(c).Grow(1).Center().Children(func() {
			ui.Text(c, "还没有任务").FontSize(13).TextColor(p.faint)
		})
		return
	}

	ui.List(c.Key("groups"), &a.list, len(rows), func(i int) {
		r := a.rows[i]
		if r.f == nil {
			a.groupRow(c, r.g)
		} else {
			a.fileRow(c, r.f)
		}
	}).Grow(1)
}

// groupRow is the header of a group: what it is, how far it got, and how it
// ended.
func (a *app) groupRow(c *ui.Context, g *group) {
	p := a.pal
	row := ui.Row(c).Margin(4, 16, 4, 16).Padding(9, 14, 9, 14).Radius(10).
		Gap(10).AlignItems(ui.Center).MinHeight(44).Background(p.panel)
	if row.Hovered() {
		row.Background(p.hover)
	}
	if row.Clicked() {
		g.open = !g.open
	}
	row.Children(func() {
		if g.open {
			ui.Icon(c, iconChevronDown).Size(12, 12).TextColor(p.faint)
		} else {
			ui.Icon(c, iconChevron).Size(12, 12).TextColor(p.faint)
		}

		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(1).Children(func() {
			ui.Text(c, filepath.Base(g.root)).FontSize(13.5).FontWeight(600).TextColor(p.text).SingleLine()
			ui.Text(c, g.root).FontSize(11.5).TextColor(p.faint).SingleLine()
		})

		ui.Text(c, fmt.Sprintf("%d/%d", g.done(), len(g.files))).FontSize(12).TextColor(p.muted)
		ui.Progress(c, g.progress()).Width(120).Label("目录进度")

		switch {
		case g.running():
			ui.Spinner(c).Size(14, 14)
		case g.finished() && g.failed() == 0:
			ui.Icon(c, iconCheck).Size(15, 15).TextColor(p.success)
		case g.finished():
			ui.Icon(c, iconCross).Size(15, 15).TextColor(p.danger)
		default:
			ui.Box(c).Size(15, 15)
		}
	})
}

// fileRow is one document of a group.
func (a *app) fileRow(c *ui.Context, f *fileItem) {
	p := a.pal
	row := ui.Row(c).Padding(3, 14, 3, 46).Gap(8).AlignItems(ui.Center).MinHeight(28)
	if row.Hovered() {
		row.Background(p.hover)
	}
	row.Children(func() {
		a.badge(c, f.format)
		ui.Text(c, f.name).FontSize(12.5).TextColor(p.text).Grow(1).Shrink(1).SingleLine()
		if f.dir != "" {
			ui.Text(c, f.dir).FontSize(11).TextColor(p.faint).SingleLine()
		}
		a.statusMark(c, f)
	})
}

// footer is the summary of the run, and where it archives what it converted.
func (a *app) footer(c *ui.Context) {
	p := a.pal
	done, skipped, failed := a.totals()
	ui.Box(c).Height(64).Children(func() {
		// Left: the settings, where the archive folder shows.
		ui.Row(c).Absolute().Left(22).Top(0).Bottom(0).AlignItems(ui.Center).Gap(10).Children(func() {
			if ui.Button(c, "设置").Clicked() {
				a.openSettings()
			}
		})

		// Middle: how the run went, centered in the window.
		ui.Column(c).Absolute().Left(0).Right(0).Top(0).Bottom(0).Center().Gap(1).IgnoreInput(true).Children(func() {
			ui.Text(c, fmt.Sprintf("成功 %d · 跳过 %d · 失败 %d", done, skipped, failed)).
				FontSize(12).TextColor(p.muted)
			if a.conv != nil && a.conv.Calibre == "" {
				ui.Text(c, "未检测到 calibre，仅支持 EPUB").FontSize(11.5).TextColor(p.warning)
			}
		})

		// Right: what the user does next.
		ui.Row(c).Absolute().Right(22).Top(0).Bottom(0).AlignItems(ui.Center).Gap(10).Children(func() {
			if ui.Button(c, "选择文件夹").Height(40).PaddingX(20).FontSize(13.5).Clicked() {
				a.pickFolders()
			}
			if ui.PrimaryButton(c, "开始转换").Height(40).PaddingX(24).FontSize(13.5).Clicked() {
				a.start()
			}
		})
	})
}

// archiveLabel is where converted originals go, as the footer shows it.
func (a *app) archiveLabel() string {
	if a.archive == "" {
		return "未设置"
	}
	return a.archive
}

// pickArchive asks where converted originals go and keeps the answer.
func (a *app) pickArchive() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:    a.win,
			Title:     "选择归档文件夹",
			Directory: true,
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.post(func() { a.setArchive(paths[0]) })
	}()
}

// pickFolders asks for directories and adds them as groups.
func (a *app) pickFolders() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:    a.win,
			Title:     "选择文件夹",
			Directory: true,
			Multiple:  true,
		})
		if err != nil || len(paths) == 0 {
			return
		}
		a.post(func() { a.scan(paths) })
	}()
}
