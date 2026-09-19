package game

import (
	_ "embed"
	"encoding/binary"
	"math"
)

const (
	opponentUpperMasks    = 1 << 6
	opponentUpperTotals   = UpperGoal + 1
	opponentLowerMasks    = 1 << 6
	opponentYahtzeeStates = 3
)

// Generated with:
//
//	go test -tags tablegen ./internal/game -run TestGenerateOpponentTables
//
//go:embed opponent_tables.bin
var opponentTableData []byte

var (
	opponentUpperValues [opponentUpperMasks][opponentUpperTotals]float64
	opponentLowerValues [opponentLowerMasks][opponentYahtzeeStates]float64
)

var opponentUpperCategoryValues = [...]float64{
	1.8813, 5.2825, 8.5693, 12.1583, 15.6874, 19.1889,
}

var opponentUpperCategoryDeviations = [...]float64{
	1.22, 2.00, 2.71, 3.29, 3.85, 4.64,
}

// The continuation estimate follows the upper/lower decomposition described
// by James Glenn: solve both sections independently, then add an estimated
// upper-bonus value. This stays small enough to ship as a 17 KB table.

func init() {
	wantedValues := opponentUpperMasks*opponentUpperTotals + opponentLowerMasks*opponentYahtzeeStates
	if len(opponentTableData) != wantedValues*4 {
		panic("invalid opponent continuation table")
	}
	offset := 0
	readValue := func() float64 {
		bits := binary.LittleEndian.Uint32(opponentTableData[offset : offset+4])
		offset += 4
		return float64(math.Float32frombits(bits))
	}
	for mask := range opponentUpperValues {
		for total := range opponentUpperValues[mask] {
			opponentUpperValues[mask][total] = readValue()
		}
	}
	for mask := range opponentLowerValues {
		for state := range opponentLowerValues[mask] {
			opponentLowerValues[mask][state] = readValue()
		}
	}
}

func opponentContinuationValue(card ScoreCard) float64 {
	upperMask := 0
	for category := Ones; category <= Sixes; category++ {
		if card.Filled[category] {
			upperMask |= 1 << category
		}
	}
	upperTotal := card.UpperTotal()
	if upperTotal > UpperGoal {
		upperTotal = UpperGoal
	}

	lowerMask := 0
	for _, category := range opponentLowerCategories {
		if card.Filled[category] {
			bit, _ := opponentLowerCategoryBit(category)
			lowerMask |= bit
		}
	}
	yahtzeeState := 0
	if card.Filled[Yahtzee] {
		yahtzeeState = 1
		if card.Values[Yahtzee] == 50 {
			yahtzeeState = 2
		}
	}

	return opponentUpperValues[upperMask][upperTotal] +
		opponentLowerValues[lowerMask][yahtzeeState] +
		opponentExpectedUpperBonus(card)
}

var opponentLowerCategories = [...]Category{
	ThreeOfAKind,
	FourOfAKind,
	FullHouse,
	SmallStraight,
	LargeStraight,
	Chance,
}

func opponentLowerCategoryBit(category Category) (int, bool) {
	for index, candidate := range opponentLowerCategories {
		if category == candidate {
			return 1 << index, true
		}
	}
	return 0, false
}

func opponentExpectedUpperBonus(card ScoreCard) float64 {
	if card.Bonus() > 0 {
		return UpperBonus
	}

	mean := float64(card.UpperTotal())
	variance := 0.0
	for category := Ones; category <= Sixes; category++ {
		if card.Filled[category] {
			continue
		}
		mean += opponentUpperCategoryValues[category]
		deviation := opponentUpperCategoryDeviations[category]
		variance += deviation * deviation
	}
	if variance == 0 {
		return 0
	}

	z := (float64(UpperGoal) - 0.5 - mean) / math.Sqrt(2*variance)
	return UpperBonus * 0.5 * math.Erfc(z)
}
