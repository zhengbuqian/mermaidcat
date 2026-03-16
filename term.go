package main

import (
	"os"

	"golang.org/x/term"
)

func makeRaw(tty *os.File) (*term.State, error) {
	return term.MakeRaw(int(tty.Fd()))
}

func restoreTerminal(tty *os.File, state *term.State) {
	term.Restore(int(tty.Fd()), state)
}
