package ui

import (
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

type Terminal struct {
	input       *os.File
	output      io.Writer
	state       *term.State
	inAlternate bool
}

func NewTerminal(input *os.File, output io.Writer) *Terminal {
	return &Terminal{input: input, output: output}
}

func (t *Terminal) Open() error {
	if !term.IsTerminal(int(t.input.Fd())) {
		return errors.New("stdin must be an interactive terminal")
	}

	state, err := term.MakeRaw(int(t.input.Fd()))
	if err != nil {
		return fmt.Errorf("enable raw terminal mode: %w", err)
	}
	t.state = state
	return nil
}

func (t *Terminal) BeginGame() error {
	if _, err := io.WriteString(t.output, "\x1b[?1049h\x1b[?25l\x1b[2J\x1b[H"); err != nil {
		return err
	}
	t.inAlternate = true
	return nil
}

func (t *Terminal) Render(frame string) error {
	if !t.inAlternate {
		return errors.New("cannot render without an active game screen")
	}
	_, err := fmt.Fprintf(t.output, "\x1b[2J\x1b[H%s", frame)
	return err
}

func (t *Terminal) Persist(frame string) error {
	if t.inAlternate {
		if err := t.leaveAlternate(); err != nil {
			return err
		}
	}
	_, err := io.WriteString(t.output, frame)
	return err
}

func (t *Terminal) Discard() error {
	if !t.inAlternate {
		return nil
	}
	return t.leaveAlternate()
}

func (t *Terminal) leaveAlternate() error {
	if _, err := io.WriteString(t.output, "\x1b[?25h\x1b[0m\x1b[?1049l"); err != nil {
		return err
	}
	t.inAlternate = false
	return nil
}

func (t *Terminal) Close() error {
	var firstErr error
	if _, err := io.WriteString(t.output, "\x1b[?25h\x1b[0m"); err != nil {
		firstErr = err
	}
	if t.inAlternate {
		if err := t.leaveAlternate(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if t.state != nil {
		if err := term.Restore(int(t.input.Fd()), t.state); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
