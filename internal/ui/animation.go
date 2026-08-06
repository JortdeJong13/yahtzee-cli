package ui

import (
	"math/rand"
	"time"

	"github.com/jortdejong/yahtzee-cli/internal/game"
)

const (
	rollAnimationFrames = 6
	rollAnimationDelay  = 150 * time.Millisecond
)

type RollAnimator struct {
	rng *rand.Rand
}

func NewRollAnimator() *RollAnimator {
	return &RollAnimator{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (a *RollAnimator) Animate(terminal *Terminal, renderer Renderer, state game.State) error {
	if allDiceLocked(state) {
		return terminal.Render(renderer.Frame(state, ""))
	}

	for frame := 0; frame < rollAnimationFrames; frame++ {
		animatedState := state
		for index := range animatedState.Dice {
			if !animatedState.Locked[index] {
				animatedState.Dice[index] = a.rng.Intn(6) + 1
			}
		}
		if err := terminal.Render(renderer.Frame(animatedState, "")); err != nil {
			return err
		}
		if frame < rollAnimationFrames-1 {
			time.Sleep(rollAnimationDelay)
		}
	}

	return terminal.Render(renderer.Frame(state, ""))
}

func allDiceLocked(state game.State) bool {
	for _, locked := range state.Locked {
		if !locked {
			return false
		}
	}
	return true
}
