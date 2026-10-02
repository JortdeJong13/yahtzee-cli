package game

import "sort"

// PlayOpponent takes one complete opponent turn. Normal and Expert evaluate
// every possible set of dice to keep across all remaining rolls, while Easy
// only looks one reroll ahead and values points earned now. Normal discounts
// future scorecard opportunities, while Expert gives them full weight. All
// difficulties choose deterministically apart from the dice rolls.
func (g *Game) PlayOpponent(observer func()) {
	if g.outcome != InProgress || g.turn != Opponent {
		return
	}

	evaluator := newOpponentEvaluator(g.scores[Opponent], g.difficulty)
	for g.rollsLeft > 0 {
		g.Roll()
		if observer != nil {
			observer()
		}

		decision := evaluator.decide(g.dice, g.rollsLeft)
		if decision.score {
			g.Score(decision.category)
			if observer != nil {
				observer()
			}
			return
		}

		g.locked = decision.keep
		if observer != nil {
			observer()
		}
	}
}

type opponentEvaluator struct {
	card         ScoreCard
	memo         map[opponentValueKey]float64
	maxLookahead int
	futureWeight float64
}

func newOpponentEvaluator(card ScoreCard, difficulty Difficulty) opponentEvaluator {
	return opponentEvaluator{
		card:         card,
		memo:         make(map[opponentValueKey]float64),
		maxLookahead: opponentMaxLookahead(difficulty),
		futureWeight: opponentFutureWeight(difficulty),
	}
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

func (e *opponentEvaluator) decide(dice Dice, rollsLeft int) opponentDecision {
	return e.rankedDecisions(dice, rollsLeft)[0]
}

func opponentMaxLookahead(difficulty Difficulty) int {
	if difficulty == Easy {
		return 1
	}
	return MaximumRolls - 1
}

func opponentFutureWeight(difficulty Difficulty) float64 {
	switch difficulty {
	case Easy:
		return 0
	case Expert:
		return 1
	default:
		return 0.4
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
		rollsLeft = e.lookahead(rollsLeft)
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
	// Group ties relative to the highest value before ordering them, so the
	// floating-point tolerance cannot make the sort comparator non-transitive.
	tied := 1
	for tied < len(decisions) && decisions[0].value-decisions[tied].value <= 1e-9 {
		tied++
	}
	sort.SliceStable(decisions[:tied], func(i, j int) bool {
		if decisions[i].score != decisions[j].score {
			return decisions[i].score
		}
		return opponentHeldCount(decisions[i].keep) > opponentHeldCount(decisions[j].keep)
	})
	return decisions
}

func opponentHeldCount(keep [DiceCount]bool) int {
	held := 0
	for _, locked := range keep {
		if locked {
			held++
		}
	}
	return held
}

func (e *opponentEvaluator) lookahead(rollsLeft int) int {
	if e.maxLookahead > 0 && rollsLeft > e.maxLookahead {
		return e.maxLookahead
	}
	return rollsLeft
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

func (e *opponentEvaluator) categoryValue(dice Dice, category Category) (float64, bool) {
	nextCard := e.card
	earned, ok := nextCard.applyScore(dice, category)
	if !ok {
		return 0, false
	}

	value := float64(earned)
	if e.futureWeight > 0 {
		// Earned bonuses belong to this move, rather than discounted future points.
		currentFuture := opponentContinuationValue(e.card) - float64(e.card.Bonus())
		nextFuture := opponentContinuationValue(nextCard) - float64(nextCard.Bonus())
		value += e.futureWeight * (nextFuture - currentFuture)
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
