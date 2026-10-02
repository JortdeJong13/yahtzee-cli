package game

func (s ScoreCard) ScoreFor(dice Dice, category Category) (int, bool) {
	if int(category) >= CategoryCount || s.Filled[category] {
		return 0, false
	}

	if isYahtzee(dice) && s.Filled[Yahtzee] {
		face := dice[0]
		upper := Category(face - 1)
		if upper <= Sixes && !s.Filled[upper] {
			if category != upper {
				return 0, false
			}
			return face * DiceCount, true
		}
		if category >= ThreeOfAKind && category <= Chance {
			return jokerScore(dice, category), true
		}
		if allLowerCategoriesFilled(s) && category <= Sixes && !s.Filled[category] {
			return 0, true
		}
		return 0, false
	}

	return scoreCategory(dice, category), true
}

// applyScore records a legal score and returns all points earned now.
func (s *ScoreCard) applyScore(dice Dice, category Category) (int, bool) {
	score, ok := s.ScoreFor(dice, category)
	if !ok {
		return 0, false
	}
	previousBonus := s.Bonus()
	extraYahtzee := isYahtzee(dice) && s.Filled[Yahtzee] && s.Values[Yahtzee] == 50
	s.Values[category] = score
	s.Filled[category] = true
	earned := score + s.Bonus() - previousBonus
	if extraYahtzee {
		s.YahtzeeBonuses++
		earned += YahtzeeBonus
	}
	return earned, true
}

func scoreCategory(dice Dice, category Category) int {
	counts := countDice(dice)
	total := diceTotal(dice)

	switch {
	case category <= Sixes:
		face := int(category) + 1
		return counts[face] * face
	case category == ThreeOfAKind:
		if hasCountAtLeast(counts, 3) {
			return total
		}
	case category == FourOfAKind:
		if hasCountAtLeast(counts, 4) {
			return total
		}
	case category == FullHouse:
		if hasFullHouse(counts) {
			return 25
		}
	case category == SmallStraight:
		if hasStraight(counts, 4) {
			return 30
		}
	case category == LargeStraight:
		if hasStraight(counts, 5) {
			return 40
		}
	case category == Yahtzee:
		if isYahtzee(dice) {
			return 50
		}
	case category == Chance:
		return total
	}
	return 0
}

func jokerScore(dice Dice, category Category) int {
	switch category {
	case FullHouse:
		return 25
	case SmallStraight:
		return 30
	case LargeStraight:
		return 40
	default:
		return diceTotal(dice)
	}
}

func allLowerCategoriesFilled(s ScoreCard) bool {
	for _, category := range Categories[6:] {
		if !s.Filled[category] {
			return false
		}
	}
	return true
}

func countDice(dice Dice) [7]int {
	var counts [7]int
	for _, die := range dice {
		counts[die]++
	}
	return counts
}

func diceTotal(dice Dice) int {
	total := 0
	for _, die := range dice {
		total += die
	}
	return total
}

func hasCountAtLeast(counts [7]int, wanted int) bool {
	for face := 1; face <= 6; face++ {
		if counts[face] >= wanted {
			return true
		}
	}
	return false
}

func hasFullHouse(counts [7]int) bool {
	var hasThree, hasTwo bool
	for face := 1; face <= 6; face++ {
		switch counts[face] {
		case 3:
			hasThree = true
		case 2:
			hasTwo = true
		}
	}
	return hasThree && hasTwo
}

func hasStraight(counts [7]int, length int) bool {
	run := 0
	for face := 1; face <= 6; face++ {
		if counts[face] > 0 {
			run++
			if run >= length {
				return true
			}
		} else {
			run = 0
		}
	}
	return false
}

func isYahtzee(dice Dice) bool {
	return countDice(dice)[dice[0]] == DiceCount
}
