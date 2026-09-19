package game

import (
	"math/rand"
	"time"
)

const (
	DiceCount     = 5
	MaximumRolls  = 3
	UpperGoal     = 63
	UpperBonus    = 35
	YahtzeeBonus  = 100
	CategoryCount = 13
)

type Player uint8

const (
	You Player = iota
	Opponent
)

func (p Player) String() string {
	if p == Opponent {
		return "Opponent"
	}
	return "You"
}

type Category uint8

const (
	Ones Category = iota
	Twos
	Threes
	Fours
	Fives
	Sixes
	ThreeOfAKind
	FourOfAKind
	FullHouse
	SmallStraight
	LargeStraight
	Yahtzee
	Chance
)

var Categories = [...]Category{
	Ones,
	Twos,
	Threes,
	Fours,
	Fives,
	Sixes,
	ThreeOfAKind,
	FourOfAKind,
	FullHouse,
	SmallStraight,
	LargeStraight,
	Yahtzee,
	Chance,
}

var categoryNames = [...]string{
	"Ones",
	"Twos",
	"Threes",
	"Fours",
	"Fives",
	"Sixes",
	"Three of a Kind",
	"Four of a Kind",
	"Full House",
	"Small Straight",
	"Large Straight",
	"Yahtzee",
	"Chance",
}

func (c Category) String() string {
	if int(c) >= 0 && int(c) < len(categoryNames) {
		return categoryNames[c]
	}
	return "Unknown"
}

type Dice [DiceCount]int

type Outcome uint8

const (
	InProgress Outcome = iota
	YouWin
	OpponentWins
	Draw
)

type ScoreCard struct {
	Values         [CategoryCount]int
	Filled         [CategoryCount]bool
	YahtzeeBonuses int
}

func (s ScoreCard) UpperTotal() int {
	total := 0
	for _, category := range Categories[:6] {
		if s.Filled[category] {
			total += s.Values[category]
		}
	}
	return total
}

func (s ScoreCard) Bonus() int {
	if s.UpperTotal() >= UpperGoal {
		return UpperBonus
	}
	return 0
}

func (s ScoreCard) Total() int {
	total := s.Bonus() + s.YahtzeeBonuses*YahtzeeBonus
	for _, category := range Categories {
		if s.Filled[category] {
			total += s.Values[category]
		}
	}
	return total
}

func (s ScoreCard) AvailableCategories(dice Dice, rolled bool) []Category {
	if !rolled {
		return nil
	}

	available := make([]Category, 0, CategoryCount)
	for _, category := range Categories {
		if _, ok := s.ScoreFor(dice, category); ok {
			available = append(available, category)
		}
	}
	return available
}

type State struct {
	Dice         Dice
	Locked       [DiceCount]bool
	Turn         Player
	RollsLeft    int
	Rolled       bool
	Scores       [2]ScoreCard
	Selected     Category
	HasSelection bool
	Outcome      Outcome
}

type Game struct {
	dice        Dice
	locked      [DiceCount]bool
	turn        Player
	rollsLeft   int
	rolled      bool
	scores      [2]ScoreCard
	selected    Category
	outcome     Outcome
	rng         *rand.Rand
	opponentRNG *rand.Rand
	difficulty  Difficulty
}

func New() *Game {
	return NewWithDifficulty(Normal)
}

func NewWithDifficulty(difficulty Difficulty) *Game {
	return NewWithSeedAndDifficulty(time.Now().UnixNano(), difficulty)
}

func NewWithSeed(seed int64) *Game {
	return NewWithSeedAndDifficulty(seed, Normal)
}

func NewWithSeedAndDifficulty(seed int64, difficulty Difficulty) *Game {
	g := &Game{
		rng:         rand.New(rand.NewSource(seed)),
		opponentRNG: rand.New(rand.NewSource(seed ^ 0x5deece66d)),
		difficulty:  difficulty,
	}
	for i := range g.dice {
		g.dice[i] = g.randomDie()
	}
	g.startTurn(You)
	return g
}

func (g *Game) State() State {
	return State{
		Dice:         g.dice,
		Locked:       g.locked,
		Turn:         g.turn,
		RollsLeft:    g.rollsLeft,
		Rolled:       g.rolled,
		Scores:       g.scores,
		Selected:     g.selected,
		HasSelection: g.rolled && g.hasOpenCategory(g.selected),
		Outcome:      g.outcome,
	}
}

func (g *Game) randomDie() int {
	return g.rng.Intn(6) + 1
}

func (g *Game) startTurn(player Player) {
	g.turn = player
	g.rollsLeft = MaximumRolls
	g.rolled = false
	g.locked = [DiceCount]bool{}
	g.selected = CategoryCount
	if player == You {
		g.selectFirstOpen()
	}
}

func (g *Game) selectFirstOpen() {
	for _, category := range Categories {
		if !g.scores[g.turn].Filled[category] {
			g.selected = category
			return
		}
	}
}

func (g *Game) hasOpenCategory(category Category) bool {
	return int(category) < CategoryCount && !g.scores[g.turn].Filled[category]
}

func (g *Game) Roll() bool {
	if g.outcome != InProgress || g.rollsLeft == 0 {
		return false
	}
	for i := range g.dice {
		if !g.locked[i] {
			g.dice[i] = g.randomDie()
		}
	}
	g.rollsLeft--
	g.rolled = true
	g.ensureValidSelection()
	return true
}

func (g *Game) ToggleLock(index int) bool {
	if g.turn != You || !g.rolled || g.rollsLeft == 0 || index < 0 || index >= DiceCount {
		return false
	}
	g.locked[index] = !g.locked[index]
	return true
}

func (g *Game) MoveSelection(delta int) bool {
	if g.turn != You || !g.rolled || delta == 0 {
		return false
	}
	available := g.scores[g.turn].AvailableCategories(g.dice, g.rolled)
	if len(available) == 0 {
		return false
	}
	current := 0
	for i, category := range available {
		if category == g.selected {
			current = i
			break
		}
	}
	next := (current + delta) % len(available)
	if next < 0 {
		next += len(available)
	}
	g.selected = available[next]
	return true
}

func (g *Game) ensureValidSelection() {
	if g.turn != You {
		return
	}
	available := g.scores[g.turn].AvailableCategories(g.dice, g.rolled)
	for _, category := range available {
		if category == g.selected {
			return
		}
	}
	if len(available) > 0 {
		g.selected = available[0]
	}
}

func (g *Game) ScoreSelected() bool {
	return g.Score(g.selected)
}

func (g *Game) Score(category Category) bool {
	if g.outcome != InProgress || !g.rolled {
		return false
	}
	card := &g.scores[g.turn]
	score, ok := card.ScoreFor(g.dice, category)
	if !ok {
		return false
	}
	bonusYahtzee := isYahtzee(g.dice) && card.Filled[Yahtzee] && card.Values[Yahtzee] == 50
	card.Values[category] = score
	card.Filled[category] = true
	if bonusYahtzee {
		card.YahtzeeBonuses++
	}

	if g.scores[You].Complete() && g.scores[Opponent].Complete() {
		g.finish()
		return true
	}
	g.startTurn(otherPlayer(g.turn))
	return true
}

func (s ScoreCard) Complete() bool {
	for _, category := range Categories {
		if !s.Filled[category] {
			return false
		}
	}
	return true
}

func (g *Game) finish() {
	yourTotal := g.scores[You].Total()
	opponentTotal := g.scores[Opponent].Total()
	switch {
	case yourTotal > opponentTotal:
		g.outcome = YouWin
	case opponentTotal > yourTotal:
		g.outcome = OpponentWins
	default:
		g.outcome = Draw
	}
}

func otherPlayer(player Player) Player {
	if player == You {
		return Opponent
	}
	return You
}
