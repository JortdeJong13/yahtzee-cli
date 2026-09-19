//go:build tablegen

package game

import (
	"encoding/binary"
	"math"
	"math/bits"
	"os"
	"testing"
)

type opponentTableTurnKey struct {
	dice      [6]uint8
	rollsLeft uint8
}

type opponentTableTurnSolver struct {
	score func(Dice) float64
	memo  map[opponentTableTurnKey]float64
}

func TestGenerateOpponentTables(t *testing.T) {
	var upper [opponentUpperMasks][opponentUpperTotals]float64
	for filled := 5; filled >= 0; filled-- {
		for mask := 0; mask < opponentUpperMasks; mask++ {
			if bits.OnesCount(uint(mask)) != filled {
				continue
			}
			for total := 0; total < opponentUpperTotals; total++ {
				currentMask := mask
				currentTotal := total
				solver := opponentTableTurnSolver{
					memo: make(map[opponentTableTurnKey]float64),
				}
				solver.score = func(dice Dice) float64 {
					best := math.Inf(-1)
					for category := Ones; category <= Sixes; category++ {
						bit := 1 << category
						if currentMask&bit != 0 {
							continue
						}
						score := scoreCategory(dice, category)
						nextTotal := currentTotal + score
						if nextTotal > UpperGoal {
							nextTotal = UpperGoal
						}
						value := float64(score) + upper[currentMask|bit][nextTotal]
						if value > best {
							best = value
						}
					}
					return best
				}
				upper[mask][total] = solver.beforeRoll()
			}
		}
	}

	var lower [opponentLowerMasks][opponentYahtzeeStates]float64
	for filled := 6; filled >= 0; filled-- {
		for mask := 0; mask < opponentLowerMasks; mask++ {
			for yahtzeeState := 0; yahtzeeState < opponentYahtzeeStates; yahtzeeState++ {
				filledCategories := bits.OnesCount(uint(mask))
				if yahtzeeState != 0 {
					filledCategories++
				}
				if filledCategories != filled {
					continue
				}
				card := opponentLowerCard(mask, yahtzeeState)
				currentMask := mask
				currentYahtzeeState := yahtzeeState
				solver := opponentTableTurnSolver{
					memo: make(map[opponentTableTurnKey]float64),
				}
				solver.score = func(dice Dice) float64 {
					best := math.Inf(-1)
					for category := ThreeOfAKind; category <= Chance; category++ {
						score, ok := card.ScoreFor(dice, category)
						if !ok {
							continue
						}
						nextYahtzeeState := currentYahtzeeState
						if category == Yahtzee {
							if score == 50 {
								nextYahtzeeState = 2
							} else {
								nextYahtzeeState = 1
							}
						}
						bit, isMasked := opponentLowerCategoryBit(category)
						if !isMasked {
							bit = 0
						}
						value := float64(score) + lower[currentMask|bit][nextYahtzeeState]
						if value > best {
							best = value
						}
					}
					return best
				}
				lower[mask][yahtzeeState] = solver.beforeRoll()
			}
		}
	}

	file, err := os.Create("opponent_tables.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	for mask := range upper {
		for total := range upper[mask] {
			if err := binary.Write(file, binary.LittleEndian, float32(upper[mask][total])); err != nil {
				t.Fatal(err)
			}
		}
	}
	for mask := range lower {
		for state := range lower[mask] {
			if err := binary.Write(file, binary.LittleEndian, float32(lower[mask][state])); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func (s *opponentTableTurnSolver) beforeRoll() float64 {
	total := 0.0
	opponentForEachRollOutcome(DiceCount, func(rolled [6]uint8, multiplicity int) {
		total += float64(multiplicity) * s.bestValue(opponentDiceFromCounts(rolled), MaximumRolls-1)
	})
	return total / float64(opponentPower(6, DiceCount))
}

func (s *opponentTableTurnSolver) bestValue(dice Dice, rollsLeft int) float64 {
	key := opponentTableTurnKey{dice: opponentDiceCounts(dice), rollsLeft: uint8(rollsLeft)}
	if value, ok := s.memo[key]; ok {
		return value
	}
	best := s.score(dice)
	if rollsLeft > 0 {
		diceCounts := opponentDiceCounts(dice)
		opponentForEachKeep(diceCounts, func(keep [6]uint8) {
			value := s.expectedAfterKeep(keep, rollsLeft)
			if value > best {
				best = value
			}
		})
	}
	s.memo[key] = best
	return best
}

func (s *opponentTableTurnSolver) expectedAfterKeep(keep [6]uint8, rollsLeft int) float64 {
	held := 0
	for _, count := range keep {
		held += int(count)
	}
	rerolls := DiceCount - held
	total := 0.0
	opponentForEachRollOutcome(rerolls, func(rolled [6]uint8, multiplicity int) {
		var next [6]uint8
		for face := range next {
			next[face] = keep[face] + rolled[face]
		}
		total += float64(multiplicity) * s.bestValue(opponentDiceFromCounts(next), rollsLeft-1)
	})
	return total / float64(opponentPower(6, rerolls))
}

func opponentLowerCard(mask, yahtzeeState int) ScoreCard {
	card := ScoreCard{}
	for category := Ones; category <= Sixes; category++ {
		card.Filled[category] = true
	}
	for _, category := range opponentLowerCategories {
		bit, _ := opponentLowerCategoryBit(category)
		card.Filled[category] = mask&bit != 0
	}
	card.Filled[Yahtzee] = yahtzeeState != 0
	if yahtzeeState == 2 {
		card.Values[Yahtzee] = 50
	}
	return card
}
