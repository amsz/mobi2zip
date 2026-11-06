package app

import (
	"context"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"mobi2zip/internal/convert"
)

// openSettings shows the settings window, opening it the first time.
func (a *app) openSettings() {
	if a.settingsWin != nil {
		a.settingsWin.Show()
		a.settingsWin.Focus()
		return
	}
	a.settingsWin = mygo.NewWindow(mygo.WindowOptions{
		Title:           "设置",
		Width:           580,
		Height:          520,
		MinWidth:        480,
		MinHeight:       380,
		StateKey:        "settings",
		AutoHideMenuBar: true,
		BackgroundColor: "light-dark(#f7f7f9, #141416)",
		Content:         ui.View(a.settingsView),
	})
	a.settingsWin.OnClosed(func() { a.settingsWin = nil })
}

func (a *app) settingsView(c *ui.Context) {
	t := *c.Theme()
	p := paletteFor(&t)
	t.Accent, t.AccentText = p.accent, p.onAccent
	c.SetTheme(&t)
	c.Root().Fill().Background(p.bg)
	c.OnShortcut(ui.Cmd, ui.KeyW, func() { a.settingsWin.Close() })

	group := func(title string, rows func()) {
		ui.Text(c, title).FontSize(12).FontWeight(600).TextColor(p.muted).Margin(14, 4, 4, 4)
		ui.Column(c).Radius(12).Background(p.panel).Border(1, p.hairline).Dividers(1, p.hairline).Children(rows)
	}

	ui.Scroll(c).Fill().Padding(16, 22, 24, 22).Gap(4).Children(func() {
		group("工具", func() {
			ui.Row(c).Padding(12, 14).Gap(14).AlignItems(ui.Center).MinHeight(56).Children(func() {
				ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, "calibre").FontSize(13).FontWeight(500).TextColor(p.text)
					ui.Text(c, "用来把 .mobi 和 .pdf 转成图片，.epub 直接读取").FontSize(11.5).TextColor(p.muted)
					ui.Text(c, a.calibreLine()).FontSize(11).TextColor(a.calibreColor()).SingleLine()
				})
				if ui.Button(c, "验证").Clicked() {
					a.verifyCalibre()
				}
				if ui.Button(c, "选择…").Clicked() {
					a.chooseCalibre()
				}
			})
		})

		group("归档", func() {
			ui.Row(c).Padding(12, 14).Gap(14).AlignItems(ui.Center).MinHeight(56).Children(func() {
				ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, "归档目录").FontSize(13).FontWeight(500).TextColor(p.text)
					ui.Text(c, "设置归档路径后，转换成功的源文件会归档到此目录").FontSize(11.5).TextColor(p.muted)
					ui.Text(c, a.archiveLabel()).FontSize(11).TextColor(p.faint).SingleLine()
				})
				if ui.Button(c, "更改…").Clicked() {
					a.pickArchive()
				}
				if a.archive != "" && ui.Button(c, "清除").Clicked() {
					a.setArchive("")
				}
			})
		})

		group("通知", func() {
			ui.Row(c).Padding(12, 14).Gap(14).AlignItems(ui.Center).MinHeight(56).Children(func() {
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, "完成后通知").FontSize(13).FontWeight(500).TextColor(p.text)
					ui.Text(c, "转换结束时用系统通知报告结果").FontSize(11.5).TextColor(p.muted)
				})
				ui.Switch(c, &a.settings.Notify)
			})
		})
	})

	// The switch writes through its pointer, so the change is saved once it
	// differs from what was last kept.
	if a.settings.Notify != a.notifyShown {
		a.notifyShown = a.settings.Notify
		a.saveSettings()
	}
}

// calibreLine is what the tools group says about calibre.
func (a *app) calibreLine() string {
	switch a.checkState {
	case checkRunning:
		return "正在验证…"
	case checkOK:
		return "✓ " + a.checkMsg
	case checkFailed:
		return "✗ " + a.checkMsg
	}
	if a.conv != nil && a.conv.Calibre != "" {
		return a.conv.Calibre
	}
	return "未找到 calibre，点击“选择…”指定 ebook-convert"
}

// calibreColor colors the line above.
func (a *app) calibreColor() ui.Color {
	switch a.checkState {
	case checkOK:
		return a.pal.success
	case checkFailed:
		return a.pal.danger
	}
	return a.pal.faint
}

// verifyCalibre asks the calibre at hand for its version, so a path that
// only exists does not pass for a working install.
func (a *app) verifyCalibre() {
	if a.conv == nil || a.conv.Calibre == "" {
		a.checkState, a.checkMsg = checkFailed, "未找到 calibre"
		return
	}
	a.checkState, a.checkMsg = checkRunning, ""
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		v, err := convert.CalibreVersion(ctx, a.conv.Calibre)
		a.post(func() {
			if err != nil {
				a.checkState, a.checkMsg = checkFailed, err.Error()
				return
			}
			a.checkState, a.checkMsg = checkOK, v
		})
	}()
}

// chooseCalibre asks for the ebook-convert to use, and takes it.
func (a *app) chooseCalibre() {
	go func() {
		path, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:    a.win,
			Title:     "选择 ebook-convert",
			Filters:   []mygo.FileFilter{{Name: "ebook-convert", Extensions: []string{"exe", "*"}}},
			Directory: false,
		})
		if err != nil || len(path) == 0 {
			return
		}
		a.post(func() {
			a.settings.CalibrePath = path[0]
			a.conv.Calibre = path[0]
			a.saveSettings()
			a.verifyCalibre()
		})
	}()
}

// saveSettings keeps the settings, and tells the user when that fails.
func (a *app) saveSettings() {
	if err := a.settings.save(); err != nil {
		a.toast = "保存设置失败：" + err.Error()
	}
}

// setArchive points the archive at dir, "" to stop moving originals, and
// keeps the answer.
func (a *app) setArchive(dir string) {
	a.archive = dir
	a.settings.Archive = dir
	a.saveSettings()
	if dir == "" {
		a.toast = "已关闭归档"
		return
	}
	a.toast = "归档目录：" + dir
}

// checkState is how the calibre check at hand stands.
type checkState int

const (
	checkIdle checkState = iota
	checkRunning
	checkOK
	checkFailed
)
