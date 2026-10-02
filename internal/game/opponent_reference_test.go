package game

import (
	"math"
	"sort"
	"testing"
)

// This reference enumerates ordered dice and physical keep masks, independently
// of the opponent's histogram enumeration and probability weights. It checks
// the current turn under the same approximate future-score model.
type referenceTurn struct {
	card   ScoreCard
	weight float64
	memo   map[referenceTurnKey]float64
}

type referenceTurnKey struct {
	dice  Dice
	rolls int
}

func (r *referenceTurn) scoreValue(dice Dice, category Category) (float64, bool) {
	score, ok := r.card.ScoreFor(dice, category)
	if !ok {
		return 0, false
	}
	next := r.card
	if isYahtzee(dice) && r.card.Filled[Yahtzee] && r.card.Values[Yahtzee] == 50 {
		next.YahtzeeBonuses++
	}
	next.Filled[category] = true
	next.Values[category] = score
	value := float64(next.Total() - r.card.Total())
	if r.weight > 0 {
		value += r.weight * (opponentContinuationValue(next) - opponentContinuationValue(r.card) - float64(next.Bonus()-r.card.Bonus()))
	}
	return value, true
}

func (r *referenceTurn) afterKeep(dice Dice, keep [DiceCount]bool, rolls int) float64 {
	total, outcomes := 0.0, 0
	var roll func(int)
	roll = func(index int) {
		if index == DiceCount {
			total += r.bestValue(dice, rolls-1)
			outcomes++
			return
		}
		if keep[index] {
			roll(index + 1)
			return
		}
		for face := 1; face <= 6; face++ {
			dice[index] = face
			roll(index + 1)
		}
	}
	roll(0)
	return total / float64(outcomes)
}

func (r *referenceTurn) bestValue(dice Dice, rolls int) float64 {
	sort.Ints(dice[:])
	key := referenceTurnKey{dice, rolls}
	if value, ok := r.memo[key]; ok {
		return value
	}
	best := math.Inf(-1)
	for _, category := range Categories {
		if value, ok := r.scoreValue(dice, category); ok {
			best = math.Max(best, value)
		}
	}
	if rolls > 0 {
		for mask := 0; mask < 1<<DiceCount; mask++ {
			var keep [DiceCount]bool
			for index := range keep {
				keep[index] = mask&(1<<index) != 0
			}
			best = math.Max(best, r.afterKeep(dice, keep, rolls))
		}
	}
	r.memo[key] = best
	return best
}

func TestOpponentAgainstOrderedDiceReference(t *testing.T) {
	for _, difficulty := range []Difficulty{Easy, Normal, Expert} {
		for _, card := range []ScoreCard{cardNearUpperBonus(Fives, Chance), cardWithOpenCategories(SmallStraight, LargeStraight)} {
			evaluator := newOpponentEvaluator(card, difficulty)
			reference := referenceTurn{card, evaluator.futureWeight, make(map[referenceTurnKey]float64)}
			dice := Dice{5, 5, 2, 3, 6}
			decision := evaluator.decide(dice, 2)
			depth := evaluator.lookahead(2)
			want := reference.bestValue(dice, depth)
			var chosen float64
			if decision.score {
				chosen, _ = reference.scoreValue(dice, decision.category)
			} else {
				chosen = reference.afterKeep(dice, decision.keep, depth)
			}
			if math.Abs(decision.value-want) > 1e-8 || math.Abs(chosen-want) > 1e-8 {
				t.Fatalf("difficulty=%d: %s, reference best=%.9f chosen=%.9f", difficulty, decisionDescription(dice, decision), want, chosen)
			}
		}
	}
}
