package game

import (
	"fmt"
	"math"
	"testing"
)

func cardWithOpenCategories(open ...Category) ScoreCard {
	var card ScoreCard
	for category := range card.Filled {
		card.Filled[category] = true
	}
	for _, category := range open {
		card.Filled[category] = false
	}
	if card.Filled[Chance] {
		card.Values[Chance] = 5
	}
	return card
}

func cardNearUpperBonus(open ...Category) ScoreCard {
	card := cardWithOpenCategories(open...)
	card.Values[Ones] = 3
	card.Values[Twos] = 6
	card.Values[Threes] = 9
	card.Values[Fours] = 12
	card.Values[Sixes] = 18
	return card
}

func heldFaces(dice Dice, keep [DiceCount]bool) [6]uint8 {
	var counts [6]uint8
	for index, locked := range keep {
		if locked {
			counts[dice[index]-1]++
		}
	}
	return counts
}

func decisionDescription(dice Dice, decision opponentDecision) string {
	if decision.score {
		return fmt.Sprintf("score %s (value %.9f)", decision.category, decision.value)
	}
	return fmt.Sprintf("keep face counts %v (value %.9f)", heldFaces(dice, decision.keep), decision.value)
}

func TestOpponentDecisions(t *testing.T) {
	for _, difficulty := range []Difficulty{Easy, Normal, Expert} {
		for _, scenario := range []struct {
			name     string
			card     ScoreCard
			dice     Dice
			rolls    int
			score    bool
			category Category
			keep     [6]uint8
		}{
			{"keep three fives", cardWithOpenCategories(Fives), Dice{5, 5, 5, 2, 3}, 2, false, 0, [6]uint8{0, 0, 0, 0, 3, 0}},
			{"keep three fives on last reroll", cardWithOpenCategories(Fives), Dice{5, 5, 5, 2, 3}, 1, false, 0, [6]uint8{0, 0, 0, 0, 3, 0}},
			{"count immediate upper bonus", cardNearUpperBonus(Fives, Chance), Dice{5, 5, 5, 6, 6}, 0, true, Fives, [6]uint8{}},
			{"break triple for a straight", cardWithOpenCategories(SmallStraight, LargeStraight), Dice{5, 5, 5, 2, 3}, 2, false, 0, [6]uint8{0, 1, 1, 0, 1, 0}},
			{"score made large straight", cardWithOpenCategories(LargeStraight), Dice{2, 3, 4, 5, 6}, 2, true, LargeStraight, [6]uint8{}},
		} {
			t.Run(fmt.Sprintf("%d/%s", difficulty, scenario.name), func(t *testing.T) {
				evaluator := newOpponentEvaluator(scenario.card, difficulty)
				decision := evaluator.decide(scenario.dice, scenario.rolls)
				if decision.score == scenario.score && (decision.score && decision.category == scenario.category || !decision.score && heldFaces(scenario.dice, decision.keep) == scenario.keep) {
					return
				}
				for rank, alternative := range evaluator.rankedDecisions(scenario.dice, scenario.rolls) {
					t.Logf("rank %d: %s", rank, decisionDescription(scenario.dice, alternative))
				}
				t.Fatalf("unexpected decision: %s", decisionDescription(scenario.dice, decision))
			})
		}
	}
}

func TestOpponentReplayedTurn(t *testing.T) {
	for _, difficulty := range []Difficulty{Easy, Normal, Expert} {
		g := NewWithSeedAndDifficulty(29, difficulty)
		g.scores[Opponent] = cardWithOpenCategories(Fives)
		g.startTurn(Opponent)
		var trace []State
		g.PlayOpponent(func() { trace = append(trace, g.State()) })
		if len(trace) < 2 || trace[0].Dice != (Dice{5, 5, 3, 2, 5}) || trace[1].Locked != ([DiceCount]bool{true, true, false, false, true}) {
			t.Fatalf("difficulty %d reproduced a bad keep: %+v", difficulty, trace)
		}
		if !g.scores[Opponent].Complete() || g.turn != You {
			t.Fatalf("difficulty %d failed to finish its turn: %+v", difficulty, g.State())
		}
	}
}

func TestOpponentScoringBonusesAndJokers(t *testing.T) {
	for _, yahtzeeScore := range []int{0, 50} {
		for _, scenario := range []struct {
			name     string
			card     ScoreCard
			category Category
			earned   int
		}{
			{"required matching upper", cardNearUpperBonus(Fives, Chance), Fives, 60},
			{"lower joker full house", cardWithOpenCategories(FullHouse), FullHouse, 25},
			{"lower joker large straight", cardWithOpenCategories(LargeStraight), LargeStraight, 40},
			{"last upper joker scores zero", cardWithOpenCategories(Ones), Ones, 0},
		} {
			t.Run(fmt.Sprintf("yahtzee%d/%s", yahtzeeScore, scenario.name), func(t *testing.T) {
				card := scenario.card
				card.Values[Yahtzee] = yahtzeeScore
				earned := scenario.earned
				if yahtzeeScore == 50 {
					earned += YahtzeeBonus
				}
				for _, difficulty := range []Difficulty{Easy, Normal, Expert} {
					evaluator := newOpponentEvaluator(card, difficulty)
					decision := evaluator.decide(Dice{5, 5, 5, 5, 5}, 0)
					if !decision.score || decision.category != scenario.category {
						t.Fatalf("difficulty %d: %s", difficulty, decisionDescription(Dice{5, 5, 5, 5, 5}, decision))
					}
				}
				g := NewWithSeed(1)
				g.scores[Opponent] = card
				g.startTurn(Opponent)
				g.dice = Dice{5, 5, 5, 5, 5}
				g.rolled = true
				if !g.Score(scenario.category) || g.scores[Opponent].Total()-card.Total() != earned {
					t.Fatalf("expected %d earned points, got %+v", earned, g.scores[Opponent])
				}
				expectedBonuses := 0
				if yahtzeeScore == 50 {
					expectedBonuses = 1
				}
				if g.scores[Opponent].YahtzeeBonuses != expectedBonuses {
					t.Fatalf("expected %d extra Yahtzees, got %+v", expectedBonuses, g.scores[Opponent])
				}
			})
		}
	}

	card := cardWithOpenCategories(Fives)
	card.Values[Ones], card.Values[Twos], card.Values[Threes], card.Values[Fours], card.Values[Sixes] = 5, 4, 12, 12, 30
	for _, difficulty := range []Difficulty{Easy, Normal, Expert} {
		evaluator := newOpponentEvaluator(card, difficulty)
		value, ok := evaluator.categoryValue(Dice{5, 5, 5, 2, 3}, Fives)
		future := opponentContinuationValue(card) - float64(card.Bonus())
		if !ok || math.Abs(value+evaluator.futureWeight*future-15) > 1e-8 {
			t.Fatalf("difficulty %d counted an already earned bonus again: %.9f", difficulty, value)
		}
	}
}

func TestOpponentOneCategoryExpectation(t *testing.T) {
	for _, card := range []ScoreCard{cardWithOpenCategories(Fives), cardNearUpperBonus(Fives)} {
		for _, difficulty := range []Difficulty{Easy, Normal, Expert} {
			evaluator := newOpponentEvaluator(card, difficulty)
			for held := 0; held <= DiceCount; held++ {
				dice := Dice{1, 1, 1, 1, 1}
				for i := 0; i < held; i++ {
					dice[i] = 5
				}
				for rolls := 0; rolls <= 2; rolls++ {
					depth := evaluator.lookahead(rolls)
					p := 1 - math.Pow(5.0/6.0, float64(depth))
					want := float64(held*5) + float64(DiceCount-held)*5*p
					if card.UpperTotal() == 48 {
						for successes := 0; successes <= DiceCount-held; successes++ {
							if held+successes >= 3 {
								want += 35 * float64(opponentChoose(DiceCount-held, successes)) * math.Pow(p, float64(successes)) * math.Pow(1-p, float64(DiceCount-held-successes))
							}
						}
					}
					got := evaluator.decide(dice, rolls).value + evaluator.futureWeight*(opponentContinuationValue(card)-float64(card.Bonus()))
					if math.Abs(got-want) > 1e-8 {
						t.Fatalf("difficulty=%d upper=%d held=%d rolls=%d: got %.9f, want %.9f", difficulty, card.UpperTotal(), held, rolls, got, want)
					}
				}
			}
		}
	}
}

func TestOpponentIllegalScorePreservesState(t *testing.T) {
	g := NewWithSeed(1)
	g.scores[Opponent] = cardNearUpperBonus(Fives, Chance)
	g.scores[Opponent].Values[Yahtzee] = 50
	g.startTurn(Opponent)
	g.dice = Dice{5, 5, 5, 5, 5}
	g.rolled = true
	before := g.State()
	if g.Score(Chance) || g.State() != before {
		t.Fatal("an illegal Joker placement changed the game state")
	}
}

func TestOpponentEnumeration(t *testing.T) {
	for rerolls := 0; rerolls <= DiceCount; rerolls++ {
		weight := 0
		factorials := [...]int{1, 1, 2, 6, 24, 120}
		opponentForEachRollOutcome(rerolls, func(counts [6]uint8, multiplicity int) {
			count := 0
			denominator := 1
			for _, value := range counts {
				count += int(value)
				denominator *= factorials[value]
			}
			if count != rerolls || multiplicity != factorials[rerolls]/denominator {
				t.Fatalf("invalid outcome: %v weight=%d", counts, multiplicity)
			}
			weight += multiplicity
		})
		if weight != int(math.Pow(6, float64(rerolls))) {
			t.Fatalf("rerolls=%d: probability weights sum to %d", rerolls, weight)
		}
	}
	checked := 0
	opponentForEachRollOutcome(DiceCount, func(counts [6]uint8, _ int) {
		dice := opponentDiceFromCounts(counts)
		opponentForEachKeep(counts, func(keep [6]uint8) {
			if got := heldFaces(dice, opponentKeepDice(dice, keep)); got != keep {
				t.Fatalf("dice=%v keep=%v mapped=%v", dice, keep, got)
			}
			checked++
		})
	})
	if checked != 4368 {
		t.Fatalf("expected 4368 keep mappings, checked %d", checked)
	}
}

func TestOpponentPermutationAndTies(t *testing.T) {
	for _, difficulty := range []Difficulty{Easy, Normal, Expert} {
		dice := Dice{2, 3, 4, 5, 5}
		evaluator := newOpponentEvaluator(ScoreCard{}, difficulty)
		want := evaluator.decide(dice, 2)
		var permute func(int)
		permute = func(index int) {
			if index == DiceCount {
				got := evaluator.decide(dice, 2)
				if got.score != want.score || got.score && got.category != want.category || !got.score && heldFaces(dice, got.keep) != heldFaces(Dice{2, 3, 4, 5, 5}, want.keep) {
					t.Fatalf("difficulty=%d dice=%v: got %s", difficulty, dice, decisionDescription(dice, got))
				}
				return
			}
			for i := index; i < DiceCount; i++ {
				dice[index], dice[i] = dice[i], dice[index]
				permute(index + 1)
				dice[index], dice[i] = dice[i], dice[index]
			}
		}
		permute(0)

		evaluator = newOpponentEvaluator(cardWithOpenCategories(FullHouse), difficulty)
		dice = Dice{5, 5, 5, 2, 3}
		decision := evaluator.decide(dice, 1)
		if decision.score || opponentHeldCount(decision.keep) != 4 || heldFaces(dice, decision.keep)[4] != 3 {
			t.Fatalf("difficulty=%d should reroll one die on equal-value full-house actions: %s", difficulty, decisionDescription(dice, decision))
		}
	}
}
