package pty

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

type PTYTerminal struct {
	Cmd       *exec.Cmd
	File      *os.File
	StartedAt time.Time
}

func (t *PTYTerminal) Read(p []byte) (int, error) {
	return t.File.Read(p)
}

func (t *PTYTerminal) Write(p []byte) (int, error) {
	return t.File.Write(p)
}

func (t *PTYTerminal) Close() error {
	if t.File == nil {
		return nil
	}
	return t.File.Close()
}

func (t *PTYTerminal) Kill() error {
	if t.Cmd == nil || t.Cmd.Process == nil {
		return nil
	}
	return t.Cmd.Process.Kill()
}

func (t *PTYTerminal) Interrupt() error {
	if t.Cmd == nil || t.Cmd.Process == nil {
		return nil
	}
	return t.Cmd.Process.Signal(syscall.SIGINT)
}

func (t *PTYTerminal) Wait() error {
	if t.Cmd == nil {
		return nil
	}
	return t.Cmd.Wait()
}
