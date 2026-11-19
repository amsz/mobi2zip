//go:build windows

package convert

import (
	"os/exec"
	"syscall"
)

// createNoWindow asks Windows for a console that no window shows.
const createNoWindow = 0x08000000

// hideWindow stops Windows from opening a console window for cmd. Without it
// a GUI app that runs a console program such as ebook-convert pops a black
// window up for as long as the program runs.
func hideWindow(cmd *exec.Cmd) {
	attr := &syscall.SysProcAttr{HideWindow: true}
	if cmd.SysProcAttr != nil {
		attr = cmd.SysProcAttr
		attr.HideWindow = true
	}
	attr.CreationFlags |= createNoWindow
	cmd.SysProcAttr = attr
}
