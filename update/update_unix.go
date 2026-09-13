//go:build !windows

package update

import "syscall"

// detachedSysProcAttr startet den Hilfsprozess in einer eigenen Session,
// damit er unabhängig vom (gleich per os.Exit beendeten) Hauptprozess
// weiterläuft.
func detachedSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
