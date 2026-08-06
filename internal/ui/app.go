package ui

import (
	"io"
	"os"
	"time"

	"github.com/jortdejong/yahtzee-cli/internal/game"
)

func Run(input *os.File, output io.Writer) error {
	terminal := NewTerminal(input, output)
	if err := terminal.Open(); err != nil {
		return err
	}
	defer terminal.Close()

	reader := NewKeyReader(input)
	renderer := NewRenderer(os.Getenv("NO_COLOR") == "")
	animator := NewRollAnimator()

	for {
		current := game.New()
		if err := terminal.BeginGame(); err != nil {
			return err
		}

		quit, err := playGame(current, reader, terminal, renderer, animator)
		if err != nil {
			return err
		}
		if quit {
			return nil
		}

		finalFrame := renderer.FinalFrame(current.State())
		if err := terminal.Persist(finalFrame); err != nil {
			return err
		}
		if err := terminal.BeginGame(); err != nil {
			return err
		}
		if err := terminal.Render(finalFrame + "   [q] quit       [r] restart\r\n"); err != nil {
			return err
		}

		for {
			key, err := reader.Read()
			if err != nil {
				return err
			}
			switch key.Rune {
			case 'q', 'Q':
				if err := terminal.Discard(); err != nil {
					return err
				}
				return nil
			case 'r', 'R':
				if err := terminal.Discard(); err != nil {
					return err
				}
				goto restart
			}
		}

	restart:
	}
}

func playGame(current *game.Game, reader *KeyReader, terminal *Terminal, renderer Renderer, animator *RollAnimator) (bool, error) {
	quitArmed := false
	for current.State().Outcome == game.InProgress {
		state := current.State()
		notice := ""
		if quitArmed {
			notice = "Press q again to quit"
		}
		if err := terminal.Render(renderer.Frame(state, notice)); err != nil {
			return false, err
		}

		if state.Turn == game.Opponent {
			var observerErr error
			previousState := current.State()
			current.PlayOpponent(func() {
				if observerErr == nil {
					currentState := current.State()
					if currentState.Rolled && currentState.RollsLeft < previousState.RollsLeft {
						observerErr = animator.Animate(terminal, renderer, currentState)
					} else {
						observerErr = terminal.Render(renderer.Frame(currentState, ""))
					}
					previousState = currentState
				}
				if observerErr == nil {
					time.Sleep(360 * time.Millisecond)
				}
			})
			if observerErr != nil {
				return false, observerErr
			}
			continue
		}

		key, err := reader.Read()
		if err != nil {
			return false, err
		}
		if quitArmed {
			if key.Kind == KeyRune && (key.Rune == 'q' || key.Rune == 'Q') {
				if err := terminal.Discard(); err != nil {
					return false, err
				}
				return true, nil
			}
			quitArmed = false
			continue
		}

		switch {
		case key.Kind == KeyRune && (key.Rune == 'q' || key.Rune == 'Q'):
			quitArmed = true
		case key.Kind == KeyRune && (key.Rune == 'r' || key.Rune == 'R'):
			if current.Roll() {
				if err := animator.Animate(terminal, renderer, current.State()); err != nil {
					return false, err
				}
			}
		case key.Kind == KeyRune && key.Rune >= '1' && key.Rune <= '5':
			current.ToggleLock(int(key.Rune - '1'))
		case key.Kind == KeyUp:
			current.MoveSelection(-1)
		case key.Kind == KeyDown:
			current.MoveSelection(1)
		case key.Kind == KeyEnter:
			current.ScoreSelected()
		}
	}

	return false, nil
}
