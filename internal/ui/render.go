package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jortdejong/yahtzee-cli/internal/game"
)

const (
	leftWidth  = 35
	rightWidth = 40
	frameWidth = leftWidth + 2 + rightWidth + 2
)

type Renderer struct {
	colors bool
}

func NewRenderer(colors bool) Renderer {
	return Renderer{colors: colors}
}

func (r Renderer) Frame(state game.State, notice string) string {
	return r.frame(state, notice, true, game.You, game.CategoryCount)
}

func (r Renderer) FrameWithScoreHighlight(state game.State, notice string, player game.Player, category game.Category) string {
	return r.frame(state, notice, true, player, category)
}

func (r Renderer) FinalFrame(state game.State) string {
	state.Locked = [game.DiceCount]bool{}
	return r.frame(state, "", false, game.You, game.CategoryCount)
}

func (r Renderer) frame(state game.State, notice string, includeControls bool, highlightedPlayer game.Player, highlightedCategory game.Category) string {
	left := r.leftColumn(state, notice)
	right := r.scoreboard(state, highlightedPlayer, highlightedCategory)
	lines := make([]string, 0, len(left))
	for i := 0; i < len(left) || i < len(right); i++ {
		leftLine := ""
		if i < len(left) {
			leftLine = left[i]
		}
		rightLine := ""
		if i < len(right) {
			rightLine = right[i]
		}
		lines = append(lines, padVisible(leftLine, leftWidth)+"  "+rightLine)
	}
	if includeControls {
		lines = append(lines, r.controls(state))
	}
	return strings.Join(lines, "\r\n") + "\r\n"
}

func (r Renderer) leftColumn(state game.State, notice string) []string {
	lines := make([]string, 0, 21)
	lines = append(lines, "")
	diceLines := r.diceBlock(state)
	lines = append(lines, diceLines...)
	lines = append(lines, "")

	if notice == "" {
		lines = append(lines, "   "+r.turnStatus(state))
	} else {
		lines = append(lines, "   "+r.style(notice, ansiAccent))
	}
	return lines
}

func (r Renderer) controls(state game.State) string {
	items := []string{
		"[r] roll",
		"[1-5] lock/unlock",
		"[↑ ↓] select score",
		"[↵] confirm",
	}
	if state.Turn != game.You || state.RollsLeft == 0 || state.Outcome != game.InProgress {
		items[0] = r.style(items[0], ansiDim)
	}
	if state.Turn != game.You || !state.Rolled || state.RollsLeft == 0 || state.Outcome != game.InProgress {
		items[1] = r.style(items[1], ansiDim)
	}
	if state.Turn != game.You || !state.Rolled || state.Outcome != game.InProgress {
		items[2] = r.style(items[2], ansiDim)
		items[3] = r.style(items[3], ansiDim)
	}

	const sidePadding = 3
	contentWidth := 0
	for _, item := range items {
		contentWidth += visibleWidth(item)
	}
	gapWidth := frameWidth - (sidePadding * 2) - contentWidth
	baseGap := gapWidth / (len(items) - 1)
	extraGaps := gapWidth % (len(items) - 1)

	var builder strings.Builder
	builder.WriteString(strings.Repeat(" ", sidePadding))
	for i, item := range items {
		builder.WriteString(item)
		if i == len(items)-1 {
			break
		}
		gap := baseGap
		if i < extraGaps {
			gap++
		}
		builder.WriteString(strings.Repeat(" ", gap))
	}
	builder.WriteString(strings.Repeat(" ", sidePadding))
	return builder.String()
}

func (r Renderer) diceBlock(state game.State) []string {
	dice := make([][]string, game.DiceCount)
	for i, value := range state.Dice {
		dice[i] = r.die(value, state.Locked[i])
	}

	lines := make([]string, 0, 18)
	appendPair := func(first, second int) {
		for row := 0; row < 5; row++ {
			lines = append(lines, "   "+dice[first][row]+strings.Repeat(" ", leftWidth-24)+dice[second][row]+"   ")
		}
		lines = append(lines, r.centeredDieLabels(state, first, second))
	}
	appendSingle := func(index int) {
		for row := 0; row < 5; row++ {
			lines = append(lines, centerVisible(dice[index][row], leftWidth))
		}
		lines = append(lines, centerVisible(r.dieLabel(state, index), leftWidth))
	}
	appendPair(0, 1)
	appendSingle(2)
	appendPair(3, 4)
	return lines
}

func (r Renderer) centeredDieLabels(state game.State, first, second int) string {
	return "   " + r.dieLabel(state, first) + strings.Repeat(" ", leftWidth-24) + r.dieLabel(state, second) + "   "
}

func (r Renderer) dieLabel(state game.State, index int) string {
	style := ""
	if state.Locked[index] {
		style = ansiCyan
	}
	return centerVisible(r.style(fmt.Sprintf("[%d]", index+1), style), 9)
}

func (r Renderer) die(value int, locked bool) []string {
	pips := [7][3]string{
		{},
		{"       ", "   ●   ", "       "},
		{" ●     ", "       ", "     ● "},
		{" ●     ", "   ●   ", "     ● "},
		{" ●   ● ", "       ", " ●   ● "},
		{" ●   ● ", "   ●   ", " ●   ● "},
		{" ●   ● ", " ●   ● ", " ●   ● "},
	}
	style := diceStyle(locked)
	return []string{
		r.style("╔═══════╗", style),
		r.style("║"+pips[value][0]+"║", style),
		r.style("║"+pips[value][1]+"║", style),
		r.style("║"+pips[value][2]+"║", style),
		r.style("╚═══════╝", style),
	}
}

func (r Renderer) scoreboard(state game.State, highlightedPlayer game.Player, highlightedCategory game.Category) []string {
	inner := make([]string, 0, 20)
	inner = append(inner, r.style(fmt.Sprintf("%-17s %7s %11s", "Category", "You", "Opponent"), ansiBold))
	inner = append(inner, r.separator())
	for _, category := range game.Categories[:6] {
		inner = append(inner, r.scoreRow(state, category, highlightedPlayer, highlightedCategory))
	}
	inner = append(inner, r.separator())
	inner = append(inner, r.bonusRow(state))
	inner = append(inner, r.separator())
	for _, category := range game.Categories[6:] {
		inner = append(inner, r.scoreRow(state, category, highlightedPlayer, highlightedCategory))
	}
	inner = append(inner, r.separator())
	inner = append(inner, r.totalRow(state))

	lines := make([]string, 0, len(inner)+2)
	lines = append(lines, "┌"+strings.Repeat("─", rightWidth)+"┐")
	for _, line := range inner {
		lines = append(lines, "│"+padVisible(line, rightWidth)+"│")
	}
	lines = append(lines, "└"+strings.Repeat("─", rightWidth)+"┘")
	return lines
}

func (r Renderer) scoreRow(state game.State, category game.Category, highlightedPlayer game.Player, highlightedCategory game.Category) string {
	your := rightAlign(r.scoreCell(state, game.You, category, highlightedPlayer, highlightedCategory), 7)
	opponent := rightAlign(r.scoreCell(state, game.Opponent, category, highlightedPlayer, highlightedCategory), 11)
	label := category.String()
	if category == highlightedCategory {
		label = r.style(label, playerColor(highlightedPlayer))
	} else if r.selectedScore(state, game.You, category) {
		label = r.style(label, ansiBold)
	}
	return fmt.Sprintf("%s %s %s", padVisible(label, 17), your, opponent)
}

func (r Renderer) scoreCell(state game.State, player game.Player, category game.Category, highlightedPlayer game.Player, highlightedCategory game.Category) string {
	card := state.Scores[player]
	if card.Filled[category] {
		style := ansiDim
		if player == highlightedPlayer && category == highlightedCategory {
			style = playerColor(player)
		}
		value := card.Values[category]
		if category == game.Yahtzee && value == 50 {
			value += card.YahtzeeBonuses * game.YahtzeeBonus
		}
		return r.style(fmt.Sprintf("%d", value), style)
	}
	if state.Turn == player && state.Rolled {
		if score, ok := card.ScoreFor(state.Dice, category); ok {
			value := fmt.Sprintf("%d", score)
			if player == game.You && r.selectedScore(state, player, category) {
				value = "→ " + value
			}
			style := r.availableScoreStyle(state, player, category)
			return r.style(value, style)
		}
	}
	return ""
}

func (r Renderer) selectedScore(state game.State, player game.Player, category game.Category) bool {
	return state.Turn == player && state.Rolled && state.HasSelection && state.Selected == category
}

func (r Renderer) availableScoreStyle(state game.State, player game.Player, category game.Category) string {
	style := playerColor(player)
	if player == game.You && r.selectedScore(state, player, category) {
		style += ansiBold
	}
	return style
}

func playerColor(player game.Player) string {
	if player == game.You {
		return ansiAccent
	}
	return ansiYellow
}

func (r Renderer) bonusRow(state game.State) string {
	your := rightAlign(r.bonusCell(state.Scores[game.You]), 7)
	opponent := rightAlign(r.bonusCell(state.Scores[game.Opponent]), 11)
	return fmt.Sprintf("%-17s %s %s", "Bonus", your, opponent)
}

func (r Renderer) bonusCell(card game.ScoreCard) string {
	value := fmt.Sprintf("%d/%d", card.UpperTotal(), game.UpperGoal)
	if card.Bonus() > 0 {
		return r.style(value, ansiBold+ansiGreen)
	}
	return value
}

func (r Renderer) totalRow(state game.State) string {
	your := rightAlign(r.style(fmt.Sprintf("%d", state.Scores[game.You].Total()), ansiBold), 7)
	opponent := rightAlign(r.style(fmt.Sprintf("%d", state.Scores[game.Opponent].Total()), ansiBold), 11)
	return fmt.Sprintf("%s %s %s", padVisible(r.style("TOTAL", ansiBold), 17), your, opponent)
}

func (r Renderer) separator() string {
	return r.style(strings.Repeat("─", rightWidth), ansiDim)
}

func (r Renderer) turnStatus(state game.State) string {
	if state.Outcome != game.InProgress {
		result := "Draw!"
		if state.Outcome == game.YouWin {
			result = "You win!"
		} else if state.Outcome == game.OpponentWins {
			result = "You lose!"
		}
		return r.style("▶ "+result, ansiAccent)
	}
	if state.Turn == game.Opponent {
		return r.style("▶ Opponent's turn", ansiBold+ansiYellow)
	}
	prefix := r.style("▶ Your turn: ", ansiBold+ansiGreen)
	if !state.Rolled {
		return prefix + r.style("Press r to roll", ansiAccent)
	}
	if state.RollsLeft == 0 {
		return prefix + r.style("select a score", ansiAccent)
	}
	rollWord := "rolls"
	if state.RollsLeft == 1 {
		rollWord = "roll"
	}
	return prefix + r.style(fmt.Sprintf("%d %s left", state.RollsLeft, rollWord), ansiAccent)
}

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	ansiAccent = "\x1b[35m"
)

func diceStyle(locked bool) string {
	if locked {
		return ansiBold + ansiCyan
	}
	return ""
}

func (r Renderer) style(text, style string) string {
	if text == "" || !r.colors || style == "" {
		return text
	}
	return style + text + ansiReset
}

func padVisible(text string, width int) string {
	visible := visibleWidth(text)
	if visible >= width {
		return text
	}
	return text + strings.Repeat(" ", width-visible)
}

func centerVisible(text string, width int) string {
	visible := visibleWidth(text)
	if visible >= width {
		return text
	}
	left := (width - visible) / 2
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", width-visible-left)
}

func rightAlign(text string, width int) string {
	visible := visibleWidth(text)
	if visible >= width {
		return text
	}
	return strings.Repeat(" ", width-visible) + text
}

func visibleWidth(text string) int {
	return utf8.RuneCountInString(stripANSI(text))
}

func stripANSI(text string) string {
	result := text
	for _, sequence := range []string{ansiReset, ansiBold, ansiDim, ansiGreen, ansiYellow, ansiCyan, ansiAccent} {
		result = strings.ReplaceAll(result, sequence, "")
	}
	return result
}
