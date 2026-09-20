package ui

import (
	"io"
	"os"
	"time"

	"github.com/JortdeJong13/yahtzee-cli/internal/game"
	"github.com/JortdeJong13/yahtzee-cli/internal/stats"
)

func Run(input *os.File, output io.Writer, difficulty game.Difficulty) error {
	terminal := NewTerminal(input, output)
	if err := terminal.Open(); err != nil {
		return err
	}
	defer terminal.Close()

	reader := NewKeyReader(input)
	renderer := NewRenderer(os.Getenv("NO_COLOR") == "", difficulty)
	animator := NewRollAnimator()

	for {
		current := game.NewWithDifficulty(difficulty)
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

		state := current.State()
		newHighScore, err := stats.Record(difficulty, state.Outcome, state.Scores[game.You].Total())
		if err != nil {
			return err
		}
		finalFrame := renderer.FinalFrame(state, newHighScore)
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
	highlightedPlayer := game.You
	highlightedCategory := game.Category(game.CategoryCount)
	hasScoreHighlight := false
	renderFrame := func(state game.State, notice string) string {
		if hasScoreHighlight {
			return renderer.FrameWithScoreHighlight(state, notice, highlightedPlayer, highlightedCategory)
		}
		return renderer.Frame(state, notice)
	}

	for current.State().Outcome == game.InProgress {
		state := current.State()
		notice := ""
		if quitArmed {
			notice = "Press q again to quit"
		}
		if err := terminal.Render(renderFrame(state, notice)); err != nil {
			return false, err
		}

		if state.Turn == game.Opponent {
			var observerErr error
			previousState := current.State()
			current.PlayOpponent(func() {
				if observerErr == nil {
					currentState := current.State()
					if currentState.Turn == game.Opponent && currentState.Rolled && !previousState.Rolled {
						hasScoreHighlight = false
					}
					if category, ok := newlyFilledScoreCategory(previousState, currentState, game.Opponent); ok {
						highlightedPlayer = game.Opponent
						highlightedCategory = category
						hasScoreHighlight = true
					}
					if currentState.Rolled && currentState.RollsLeft < previousState.RollsLeft {
						observerErr = animator.Animate(terminal, renderer, currentState)
					} else {
						observerErr = terminal.Render(renderFrame(currentState, ""))
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
				hasScoreHighlight = false
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
			before := current.State()
			if current.ScoreSelected() {
				if category, ok := newlyFilledScoreCategory(before, current.State(), game.You); ok && current.State().Outcome == game.InProgress {
					highlightedPlayer = game.You
					highlightedCategory = category
					hasScoreHighlight = true
					if err := terminal.Render(renderFrame(current.State(), "")); err != nil {
						return false, err
					}
					time.Sleep(500 * time.Millisecond)
				}
			}
		}
	}

	return false, nil
}

func newlyFilledScoreCategory(before, after game.State, player game.Player) (game.Category, bool) {
	for _, category := range game.Categories {
		if !before.Scores[player].Filled[category] && after.Scores[player].Filled[category] {
			return category, true
		}
	}
	return 0, false
}
