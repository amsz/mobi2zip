//go:build !windows

package convert

import "os/exec"

// hideWindow does nothing away from Windows, where a child program never
// opens a window of its own.
func hideWindow(cmd *exec.Cmd) {}
