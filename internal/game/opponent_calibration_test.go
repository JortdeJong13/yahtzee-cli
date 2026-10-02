//go:build botcheck

package game

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

var (
	calibrationGames = flag.Int("opponent-games", 2000, "games per opponent profile")
	calibrationSeed  = flag.Int64("opponent-seed", 10001, "first calibration game seed")
	calibrationSweep = flag.Bool("opponent-sweep", false, "also measure full-turn future weights from zero to one")
)

type calibrationProfile struct {
	name       string
	difficulty Difficulty
	weight     float64
	custom     bool
}

type calibrationResult struct {
	scores []int
	mean   float64
	margin float64
}

// Intn(6) consumes Int31n(6), which accepts these small Int31 values without
// rejection. This supplies known faces to Game.Roll without changing production
// dice generation. Unlocked indices receive their own scenario's dice values.
type calibrationDiceSource struct {
	faces []int
	next  int
}

func (s *calibrationDiceSource) Seed(int64) { s.next = 0 }

func (s *calibrationDiceSource) Int63() int64 {
	face := s.faces[s.next]
	s.next++
	return int64(face-1) << 32
}

func supplyCalibrationRoll(g *Game, dice Dice) {
	source := &calibrationDiceSource{}
	for index, face := range dice {
		if !g.locked[index] {
			source.faces = append(source.faces, face)
		}
	}
	g.rng = rand.New(source)
}

func playCalibrationTurn(g *Game, profile calibrationProfile, rolls [MaximumRolls]Dice) {
	if profile.custom {
		evaluator := newOpponentEvaluator(g.scores[Opponent], Expert)
		evaluator.futureWeight = profile.weight
		for roll := 0; roll < MaximumRolls; roll++ {
			supplyCalibrationRoll(g, rolls[roll])
			g.Roll()
			decision := evaluator.decide(g.dice, g.rollsLeft)
			if decision.score {
				g.Score(decision.category)
				return
			}
			g.locked = decision.keep
		}
		panic("calibration turn did not score")
	}
	supplyCalibrationRoll(g, rolls[0])
	observedRollsLeft := MaximumRolls
	g.PlayOpponent(func() {
		if g.turn != Opponent || g.outcome != InProgress || g.rollsLeft == 0 {
			return
		}
		if g.rollsLeft == observedRollsLeft {
			supplyCalibrationRoll(g, rolls[MaximumRolls-g.rollsLeft])
		}
		observedRollsLeft = g.rollsLeft
	})
}

func meanAndMargin(values []float64) (float64, float64) {
	mean := 0.0
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, value := range values {
		variance += (value - mean) * (value - mean)
	}
	variance /= float64(len(values) - 1)
	return mean, 1.96 * math.Sqrt(variance/float64(len(values)))
}

func TestOpponentCalibration(t *testing.T) {
	if *calibrationGames < 2 {
		t.Fatal("opponent-games must be at least two")
	}
	profiles := []calibrationProfile{{name: "Easy", difficulty: Easy}, {name: "Normal", difficulty: Normal}, {name: "Expert", difficulty: Expert}}
	if *calibrationSweep {
		for _, weight := range []float64{0, 0.25, 0.4, 0.5, 0.75, 1} {
			profiles = append(profiles, calibrationProfile{name: fmt.Sprintf("future weight %g", weight), weight: weight, custom: true})
		}
	}
	results := make([]calibrationResult, len(profiles))
	t.Run("profiles", func(t *testing.T) {
		for index, profile := range profiles {
			t.Run(profile.name, func(t *testing.T) {
				t.Parallel()
				scores := make([]int, *calibrationGames)
				values := make([]float64, *calibrationGames)
				upperBonus, yahtzee, extraYahtzees, zeroed := 0, 0, 0, 0
				var thinking time.Duration
				for game := range scores {
					seed := *calibrationSeed + int64(game)
					g := NewWithSeedAndDifficulty(seed, profile.difficulty)
					// Complete the unused human card to finish solo games normally.
					for category := range g.scores[You].Filled {
						g.scores[You].Filled[category] = true
					}
					for turn := 0; turn < CategoryCount; turn++ {
						rng := rand.New(rand.NewSource(seed*100 + int64(turn)))
						var rolls [MaximumRolls]Dice
						for roll := range rolls {
							for die := range rolls[roll] {
								rolls[roll][die] = rng.Intn(6) + 1
							}
						}
						g.startTurn(Opponent)
						before := g.scores[Opponent]
						start := time.Now()
						playCalibrationTurn(g, profile, rolls)
						thinking += time.Since(start)
						newlyFilled := 0
						for category := range before.Filled {
							if !before.Filled[category] && g.scores[Opponent].Filled[category] {
								newlyFilled++
							}
						}
						if newlyFilled != 1 {
							t.Fatalf("seed=%d turn=%d filled %d categories", seed, turn, newlyFilled)
						}
					}
					card := g.scores[Opponent]
					if !card.Complete() || g.outcome == InProgress {
						t.Fatalf("seed=%d did not finish: %+v", seed, g.State())
					}
					scores[game] = card.Total()
					values[game] = float64(card.Total())
					if card.Bonus() > 0 {
						upperBonus++
					}
					if card.Values[Yahtzee] == 50 {
						yahtzee++
					}
					extraYahtzees += card.YahtzeeBonuses
					for _, score := range card.Values {
						if score == 0 {
							zeroed++
						}
					}
				}
				mean, margin := meanAndMargin(values)
				results[index] = calibrationResult{scores, mean, margin}
				sorted := append([]int(nil), scores...)
				sort.Ints(sorted)
				n := len(sorted)
				t.Logf("seed=%d games=%d mean=%.3f CI95=±%.3f median=%d p10=%d p90=%d upperBonus=%.1f%% yahtzee=%.1f%% extraYahtzees=%d zeroedPerGame=%.2f msPerTurn=%.3f", *calibrationSeed, n, mean, margin, sorted[n/2], sorted[n/10], sorted[n*9/10], 100*float64(upperBonus)/float64(n), 100*float64(yahtzee)/float64(n), extraYahtzees, float64(zeroed)/float64(n), float64(thinking)/float64(time.Millisecond)/float64(n*CategoryCount))
			})
		}
	})
	if *calibrationGames >= 1000 {
		bands := [][2]float64{{205, 220}, {230, 240}, {240, 265}}
		for index, band := range bands {
			result := results[index]
			if result.mean+result.margin < band[0] || result.mean-result.margin > band[1] {
				t.Errorf("%s mean %.3f ± %.3f is outside target band %v", profiles[index].name, result.mean, result.margin, band)
			}
		}
	}
	for index := 1; index < 3; index++ {
		differences := make([]float64, *calibrationGames)
		for game := range differences {
			differences[game] = float64(results[index].scores[game] - results[index-1].scores[game])
		}
		mean, margin := meanAndMargin(differences)
		t.Logf("paired %s minus %s: %.3f ± %.3f", profiles[index].name, profiles[index-1].name, mean, margin)
		if *calibrationGames >= 1000 && mean-margin <= 0 {
			t.Errorf("%s did not clearly outperform %s", profiles[index].name, profiles[index-1].name)
		}
	}
}

func TestCalibrationDiceSource(t *testing.T) {
	g := NewWithSeed(1)
	g.dice = Dice{1, 2, 3, 4, 5}
	g.locked = [DiceCount]bool{true, false, true, false, false}
	supplyCalibrationRoll(g, Dice{6, 5, 4, 3, 2})
	g.Roll()
	if g.dice != (Dice{1, 5, 3, 3, 2}) {
		t.Fatalf("held indices changed the scenario's dice: %v", g.dice)
	}
}
