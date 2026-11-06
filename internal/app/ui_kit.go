package app

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

// palette holds the app's colors for the light and the dark appearance.
type palette struct {
	dark     bool
	bg       ui.Color // the window's background
	panel    ui.Color // cards and the list's rows
	text     ui.Color
	muted    ui.Color
	faint    ui.Color
	hairline ui.Color
	hover    ui.Color
	press    ui.Color
	accent   ui.Color
	onAccent ui.Color
	success  ui.Color
	danger   ui.Color
	warning  ui.Color
}

func paletteFor(t *ui.Theme) palette {
	if t.Dark {
		return palette{
			dark:     true,
			bg:       ui.Hex("#141416"),
			panel:    ui.Hex("#1c1c1f"),
			text:     ui.Hex("#f5f5f7"),
			muted:    ui.RGBA(235, 235, 245, 0.56),
			faint:    ui.RGBA(235, 235, 245, 0.3),
			hairline: ui.RGBA(255, 255, 255, 0.08),
			hover:    ui.RGBA(255, 255, 255, 0.06),
			press:    ui.RGBA(255, 255, 255, 0.1),
			accent:   ui.Oklch(0.66, 0.2, 255),
			onAccent: ui.Hex("#ffffff"),
			success:  ui.Oklch(0.72, 0.17, 150),
			danger:   ui.Oklch(0.65, 0.2, 25),
			warning:  ui.Oklch(0.78, 0.15, 80),
		}
	}
	return palette{
		bg:       ui.Hex("#f7f7f9"),
		panel:    ui.Hex("#ffffff"),
		text:     ui.Hex("#1d1d1f"),
		muted:    ui.RGBA(60, 60, 67, 0.62),
		faint:    ui.RGBA(60, 60, 67, 0.32),
		hairline: ui.RGBA(0, 0, 0, 0.08),
		hover:    ui.RGBA(0, 0, 0, 0.04),
		press:    ui.RGBA(0, 0, 0, 0.08),
		accent:   ui.Oklch(0.6, 0.21, 258),
		onAccent: ui.Hex("#ffffff"),
		success:  ui.Oklch(0.62, 0.16, 150),
		danger:   ui.Oklch(0.58, 0.21, 27),
		warning:  ui.Oklch(0.72, 0.15, 80),
	}
}

// badge is the little label naming a file's format, in a column of its own
// so that the names of a list line up.
func (a *app) badge(c *ui.Context, text string) ui.Element {
	p := a.pal
	b := ui.Box(c).Width(46).Height(18).Radius(5).Background(p.hover).Center()
	b.Children(func() {
		ui.Text(c, text).FontSize(10.5).FontWeight(700).TextColor(p.muted).SingleLine()
	})
	return b
}

// statusMark is what a file's row says about it at its end: how far it got
// while it runs, and how it ended.
func (a *app) statusMark(c *ui.Context, f *fileItem) {
	p := a.pal
	switch f.state {
	case fileRunning:
		ui.Text(c, fmtPercentage(f.pct)).FontSize(11.5).TextColor(p.accent)
	case fileDone:
		ui.Icon(c, iconCheck).Size(14, 14).TextColor(p.success).Tooltip(f.zip)
	case fileFailed:
		ui.Icon(c, iconCross).Size(14, 14).TextColor(p.danger).Tooltip(f.err)
	case fileSkipped:
		ui.Icon(c, iconWarningMark).Size(14, 14).TextColor(p.warning).Tooltip(f.err)
	default:
		ui.Text(c, "等待").FontSize(11.5).TextColor(p.faint)
	}
}

// fmtPercentage writes a share of the work, for a running file.
func fmtPercentage(pct int) string {
	return strconv.Itoa(pct) + "%"
}
