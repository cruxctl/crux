package pty

import (
	"os"

	"golang.org/x/sys/unix"
)

// SaveTerminalState captures the current termios for the given fd.
func SaveTerminalState(fd int) (*unix.Termios, error) {
	return unix.IoctlGetTermios(fd, unix.TCGETS)
}

// RestoreTerminalState restores termios for the given fd.
func RestoreTerminalState(fd int, state *unix.Termios) error {
	return unix.IoctlSetTermios(fd, unix.TCSETS, state)
}

// SetRawMode puts the terminal into raw mode.
func SetRawMode(fd int) (*unix.Termios, error) {
	old, err := SaveTerminalState(fd)
	if err != nil {
		return nil, err
	}
	newState := *old
	// Disable echo, canonical mode, and signals.
	newState.Lflag &^= unix.ECHO | unix.ICANON | unix.ISIG | unix.IEXTEN
	newState.Iflag &^= unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
	newState.Cflag |= unix.CS8
	newState.Oflag &^= unix.OPOST
	newState.Cc[unix.VMIN] = 1
	newState.Cc[unix.VTIME] = 0
	if err := RestoreTerminalState(fd, &newState); err != nil {
		return old, err
	}
	return old, nil
}

// IsTerminal reports whether fd is a terminal.
func IsTerminal(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	return err == nil
}

// StdinTerminalState returns the terminal state for os.Stdin if it's a tty.
func StdinTerminalState() (*unix.Termios, error) {
	if !IsTerminal(int(os.Stdin.Fd())) {
		return nil, nil
	}
	return SaveTerminalState(int(os.Stdin.Fd()))
}
