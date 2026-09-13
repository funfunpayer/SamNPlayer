//go:build windows

package update

import "syscall"

// detachedSysProcAttr sorgt dafür, dass der Hilfsprozess unabhängig vom
// aktuellen Prozess weiterläuft (kein Konsolenfenster, eigene Prozessgruppe),
// damit os.Exit() im Hauptprozess ihn nicht mit runterreißt.
func detachedSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x08000000, // + CREATE_NO_WINDOW
	}
}
