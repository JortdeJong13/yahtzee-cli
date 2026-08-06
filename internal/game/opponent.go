package game

import "math"

// PlayOpponent takes one complete expected-value turn. It evaluates every
// possible set of dice to keep and every possible result of the next roll.
// The evaluator optimizes one roll ahead while using category values as an
// estimate of the opportunity cost of consuming each open category.
func (g *Game) PlayOpponent(observer func()) {
	if g.outcome != InProgress || g.turn != Opponent {
		return
	}
	if observer != nil {
		observer()
	}

	evaluator := opponentEvaluator{
		card: g.scores[Opponent],
	}
	var category Category
	for g.rollsLeft > 0 {
		g.Roll()
		if observer != nil {
			observer()
		}

		decision := evaluator.decide(g.dice, g.rollsLeft)
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
}

type opponentDecision struct {
	score    bool
	category Category
	keep     [DiceCount]bool
}

// These are approximate category values under strong solitaire play. They
// give the evaluator a lightweight estimate of what is lost by consuming an
// open category now instead of using it on a later turn.
var opponentCategoryValues = [...]float64{
	1.8813, 5.2825, 8.5693, 12.1583, 15.6874, 19.1889,
	21.6614, 13.0977, 22.5918, 29.4612, 32.7113, 16.8683, 22.0091,
}

func (e *opponentEvaluator) decide(dice Dice, rollsLeft int) opponentDecision {
	category, scoreValue := e.bestCategory(dice)
	if rollsLeft == 0 {
		return opponentDecision{score: true, category: category}
	}

	keepCounts, rollValue := e.bestKeep(dice)
	if scoreValue >= rollValue {
		return opponentDecision{score: true, category: category}
	}
	return opponentDecision{keep: opponentKeepDice(dice, keepCounts)}
}

func (e *opponentEvaluator) bestKeep(dice Dice) ([6]uint8, float64) {
	bestKeep := [6]uint8{}
	bestValue := math.Inf(-1)
	diceCounts := opponentDiceCounts(dice)
	opponentForEachKeep(diceCounts, func(keep [6]uint8) {
		value := e.expectedAfterKeep(keep)
		if value > bestValue {
			bestKeep = keep
			bestValue = value
		}
	})
	return bestKeep, bestValue
}

func (e *opponentEvaluator) expectedAfterKeep(keep [6]uint8) float64 {
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
		_, value := e.bestCategory(opponentDiceFromCounts(nextCounts))
		total += float64(multiplicity) * value
	})
	return total / float64(outcomes)
}

func (e *opponentEvaluator) bestCategory(dice Dice) (Category, float64) {
	bestCategory := Categories[0]
	bestValue := math.Inf(-1)
	for _, category := range Categories {
		score, ok := e.categoryValue(dice, category)
		if ok && score > bestValue {
			bestCategory = category
			bestValue = score
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

	value := float64(score) - opponentCategoryValues[category]
	value += float64(nextCard.Bonus() - e.card.Bonus())
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
