package game

import (
	"math/rand"
	"sort"
)

// PlayOpponent takes one complete expected-value turn. It evaluates every
// possible set of dice to keep across all remaining rolls, then estimates the
// rest of the game from precomputed upper- and lower-section continuations.
func (g *Game) PlayOpponent(observer func()) {
	if g.outcome != InProgress || g.turn != Opponent {
		return
	}

	evaluator := opponentEvaluator{
		card: g.scores[Opponent],
		memo: make(map[opponentValueKey]float64),
	}
	var category Category
	for g.rollsLeft > 0 {
		g.Roll()
		if observer != nil {
			observer()
		}

		decision := evaluator.decide(g.dice, g.rollsLeft, g.difficulty, g.opponentRNG)
		if decision.score {
			category = decision.category
			break
		}

		g.locked = decision.keep
		if observer != nil {
			observer()
		}
	}

	if !g.rolled {
		return
	}
	if g.rollsLeft == 0 || category >= CategoryCount || g.scores[Opponent].Filled[category] {
		category, _ = evaluator.bestCategory(g.dice)
	}
	g.Score(category)
	if observer != nil {
		observer()
	}
}

type opponentEvaluator struct {
	card ScoreCard
	memo map[opponentValueKey]float64
}

type opponentDecision struct {
	score    bool
	category Category
	keep     [DiceCount]bool
	value    float64
}

type opponentValueKey struct {
	dice      [6]uint8
	rollsLeft uint8
}

func (e *opponentEvaluator) decide(dice Dice, rollsLeft int, difficulty Difficulty, rng *rand.Rand) opponentDecision {
	decisions := e.rankedDecisions(dice, rollsLeft)
	best := decisions[0]

	mistakeChance, maximumLoss := difficultyLimits(difficulty)
	if mistakeChance == 0 || rng.Float64() >= mistakeChance {
		return best
	}

	alternatives := make([]opponentDecision, 0, len(decisions)-1)
	for _, decision := range decisions[1:] {
		if best.value-decision.value > maximumLoss {
			break
		}
		alternatives = append(alternatives, decision)
	}
	if len(alternatives) == 0 {
		return best
	}
	return alternatives[rng.Intn(len(alternatives))]
}

func difficultyLimits(difficulty Difficulty) (mistakeChance, maximumLoss float64) {
	switch difficulty {
	case Easy:
		return 0.35, 5
	case Expert:
		return 0, 0
	default:
		return 0.18, 3
	}
}

func (e *opponentEvaluator) rankedDecisions(dice Dice, rollsLeft int) []opponentDecision {
	decisions := make([]opponentDecision, 0, CategoryCount+32)
	for _, category := range Categories {
		if value, ok := e.categoryValue(dice, category); ok {
			decisions = append(decisions, opponentDecision{
				score:    true,
				category: category,
				value:    value,
			})
		}
	}

	if rollsLeft > 0 {
		diceCounts := opponentDiceCounts(dice)
		opponentForEachKeep(diceCounts, func(keep [6]uint8) {
			decisions = append(decisions, opponentDecision{
				keep:  opponentKeepDice(dice, keep),
				value: e.expectedAfterKeep(keep, rollsLeft),
			})
		})
	}

	sort.SliceStable(decisions, func(i, j int) bool {
		return decisions[i].value > decisions[j].value
	})
	return decisions
}

func (e *opponentEvaluator) expectedAfterKeep(keep [6]uint8, rollsLeft int) float64 {
	held := 0
	for _, count := range keep {
		held += int(count)
	}
	rerolls := DiceCount - held
	outcomes := opponentPower(6, rerolls)
	total := 0.0
	opponentForEachRollOutcome(rerolls, func(rolled [6]uint8, multiplicity int) {
		var nextCounts [6]uint8
		for face := range nextCounts {
			nextCounts[face] = keep[face] + rolled[face]
		}
		total += float64(multiplicity) * e.bestValue(opponentDiceFromCounts(nextCounts), rollsLeft-1)
	})
	return total / float64(outcomes)
}

func (e *opponentEvaluator) bestValue(dice Dice, rollsLeft int) float64 {
	key := opponentValueKey{
		dice:      opponentDiceCounts(dice),
		rollsLeft: uint8(rollsLeft),
	}
	if value, ok := e.memo[key]; ok {
		return value
	}
	value := e.rankedDecisions(dice, rollsLeft)[0].value
	e.memo[key] = value
	return value
}

func (e *opponentEvaluator) bestCategory(dice Dice) (Category, float64) {
	bestCategory := Categories[0]
	bestValue := 0.0
	found := false
	for _, category := range Categories {
		score, ok := e.categoryValue(dice, category)
		if ok && (!found || score > bestValue) {
			bestCategory = category
			bestValue = score
			found = true
		}
	}
	return bestCategory, bestValue
}

func (e *opponentEvaluator) categoryValue(dice Dice, category Category) (float64, bool) {
	score, ok := e.card.ScoreFor(dice, category)
	if !ok {
		return 0, false
	}

	nextCard := e.card
	nextCard.Values[category] = score
	nextCard.Filled[category] = true

	value := float64(score)
	value += opponentContinuationValue(nextCard) - opponentContinuationValue(e.card)
	if isYahtzee(dice) && e.card.Filled[Yahtzee] && e.card.Values[Yahtzee] == 50 {
		value += YahtzeeBonus
	}
	return value, true
}

func opponentForEachKeep(diceCounts [6]uint8, visit func([6]uint8)) {
	var keep [6]uint8
	var visitKeeps func(int)
	visitKeeps = func(face int) {
		if face == len(keep) {
			visit(keep)
			return
		}
		for count := uint8(0); count <= diceCounts[face]; count++ {
			keep[face] = count
			visitKeeps(face + 1)
		}
	}
	visitKeeps(0)
}

func opponentForEachRollOutcome(rerolls int, visit func([6]uint8, int)) {
	var rolled [6]uint8
	var visitFaces func(int, int, int)
	visitFaces = func(face, remaining, multiplicity int) {
		if face == len(rolled)-1 {
			rolled[face] = uint8(remaining)
			visit(rolled, multiplicity)
			return
		}
		for count := 0; count <= remaining; count++ {
			rolled[face] = uint8(count)
			visitFaces(face+1, remaining-count, multiplicity*opponentChoose(remaining, count))
		}
	}
	visitFaces(0, rerolls, 1)
}

func opponentKeepDice(dice Dice, keepCounts [6]uint8) [DiceCount]bool {
	keep := [DiceCount]bool{}
	for index, face := range dice {
		faceIndex := face - 1
		if keepCounts[faceIndex] > 0 {
			keep[index] = true
			keepCounts[faceIndex]--
		}
	}
	return keep
}

func opponentDiceCounts(dice Dice) [6]uint8 {
	var counts [6]uint8
	for _, face := range dice {
		counts[face-1]++
	}
	return counts
}

func opponentDiceFromCounts(counts [6]uint8) Dice {
	var dice Dice
	index := 0
	for face, count := range counts {
		for i := uint8(0); i < count; i++ {
			dice[index] = face + 1
			index++
		}
	}
	return dice
}

func opponentPower(base, exponent int) int {
	value := 1
	for i := 0; i < exponent; i++ {
		value *= base
	}
	return value
}

func opponentChoose(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	if k > n-k {
		k = n - k
	}
	value := 1
	for i := 1; i <= k; i++ {
		value = value * (n - k + i) / i
	}
	return value
}
