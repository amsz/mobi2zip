package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"mobi2zip/internal/convert"
)

// app is the state the window shows. It is read and changed on the main
// thread: by the view, and by goroutines through post.
type app struct {
	win *mygo.Window
	pal palette

	zone ui.Handle
	list ui.ListState

	groups   []*group
	rows     []listRow
	nextID   int
	busy     bool
	toast    string
	archive  string // the folder converted originals move to, once set
	settings settings
	conv     *convert.Converter

	settingsWin *mygo.Window
	notifyShown bool       // the switch as last kept, to save it when it moves
	checkState  checkState // how the calibre check at hand stands
	checkMsg    string
}

// Run starts the app and shows the window.
func Run() error {
	a := &app{conv: &convert.Converter{Calibre: convert.FindCalibre()}}
	mygo.App.WhenReady(func() {
		a.settings = loadSettings()
		a.archive = a.settings.Archive
		a.notifyShown = a.settings.Notify
		if a.settings.CalibrePath != "" {
			a.conv.Calibre = a.settings.CalibrePath
		}
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:           "漫画解包",
			Width:           820,
			Height:          640,
			MinWidth:        640,
			MinHeight:       460,
			StateKey:        "main",
			TitleBarStyle:   mygo.TitleBarHidden,
			TitleBarHeight:  titleBarHeight,
			BackgroundColor: "light-dark(#f7f7f9, #141416)",
			Content:         ui.View(a.view),
		})
		// A development aid: MOBI2ZIP_DEMO names a directory to list at
		// start, so the views can be looked at without dropping it.
		a.devSeed()
	})
	return mygo.App.Run()
}

// post runs fn on the main thread and redraws the windows. Without a window,
// as in tests, it drops fn.
func (a *app) post(fn func()) {
	if a.win == nil {
		return
	}
	a.win.Update(func() {
		fn()
		a.redraw()
	})
}

// redraw asks for the next frame, in every window the app has open.
func (a *app) redraw() {
	if a.win != nil {
		a.win.Invalidate()
	}
	if a.settingsWin != nil {
		a.settingsWin.Invalidate()
	}
}
