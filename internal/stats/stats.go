package stats

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/JortdeJong13/yahtzee-cli/internal/game"
)

const (
	currentVersion = 1
	innerWidth     = 56
	ansiReset      = "\x1b[0m"
	ansiDim        = "\x1b[2m"
)

type DifficultyStats struct {
	Played       int `json:"played"`
	Wins         int `json:"wins"`
	Draws        int `json:"draws"`
	HighestScore int `json:"highest_score"`
}

type file struct {
	Version      int                        `json:"version"`
	Difficulties map[string]DifficultyStats `json:"difficulties"`
}

var difficultyOrder = [...]game.Difficulty{game.Easy, game.Normal, game.Expert}

func Record(difficulty game.Difficulty, outcome game.Outcome, score int) (bool, error) {
	name, ok := difficultyName(difficulty)
	if !ok {
		return false, fmt.Errorf("record stats: unknown difficulty %d", difficulty)
	}
	if score < 0 {
		return false, fmt.Errorf("record stats: invalid score %d", score)
	}

	data, err := load()
	if err != nil {
		return false, fmt.Errorf("record stats: %w", err)
	}

	newHighScore := totalPlayed(data) > 0 && score > highestScore(data)
	entry := data.Difficulties[name]
	entry.Played++
	switch outcome {
	case game.YouWin:
		entry.Wins++
	case game.Draw:
		entry.Draws++
	case game.OpponentWins:
	default:
		return false, errors.New("record stats: game is not complete")
	}
	if score > entry.HighestScore {
		entry.HighestScore = score
	}
	data.Difficulties[name] = entry

	if err := save(data); err != nil {
		return false, fmt.Errorf("record stats: %w", err)
	}
	return newHighScore, nil
}

func Print(w io.Writer, colors bool) error {
	data, err := load()
	if err != nil {
		return fmt.Errorf("read stats: %w", err)
	}

	totalPlayed := totalPlayed(data)
	highestScore := highestScore(data)

	var builder strings.Builder
	fmt.Fprintf(&builder, "\nGames played: %d\nHigh score: %d\n\n", totalPlayed, highestScore)
	builder.WriteString("┌" + strings.Repeat("─", innerWidth) + "┐\n")
	builder.WriteString("│" + fmt.Sprintf(" %-12s %7s %6s %8s %7s  %8s ", "Difficulty", "Played", "Wins", "Losses", "Draws", "Win rate") + "│\n")
	separator := strings.Repeat("─", innerWidth)
	if colors {
		separator = ansiDim + separator + ansiReset
	}
	builder.WriteString("│" + separator + "│\n")
	for _, difficulty := range difficultyOrder {
		name := difficultyString(difficulty)
		entry := data.Difficulties[name]
		losses := entry.Played - entry.Wins - entry.Draws
		winRate := 0
		if entry.Played > 0 {
			winRate = (entry.Wins*100 + entry.Played/2) / entry.Played
		}
		winRateText := fmt.Sprintf("%d%%", winRate)
		row := fmt.Sprintf(" %-12s %7d %6d %8d %7d  %8s ", difficultyLabel(difficulty), entry.Played, entry.Wins, losses, entry.Draws, winRateText)
		builder.WriteString("│" + row + "│\n")
	}
	builder.WriteString("└" + strings.Repeat("─", innerWidth) + "┘\n")

	_, err = io.WriteString(w, builder.String())
	return err
}

func load() (file, error) {
	path, err := statsPath()
	if err != nil {
		return file{}, err
	}

	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return newFile(), nil
	}
	if err != nil {
		return file{}, err
	}

	data := file{}
	if err := json.Unmarshal(contents, &data); err != nil {
		return file{}, err
	}
	if data.Version != currentVersion {
		return file{}, fmt.Errorf("unsupported stats version %d", data.Version)
	}
	if err := validate(data); err != nil {
		return file{}, err
	}
	for _, difficulty := range difficultyOrder {
		name := difficultyString(difficulty)
		if _, ok := data.Difficulties[name]; !ok {
			data.Difficulties[name] = DifficultyStats{}
		}
	}
	return data, nil
}

func save(data file) error {
	path, err := statsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(path), ".stats-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return err
	}

	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func validate(data file) error {
	if data.Difficulties == nil {
		return errors.New("stats file has no difficulties")
	}
	for name, entry := range data.Difficulties {
		if entry.Played < 0 || entry.Wins < 0 || entry.Draws < 0 || entry.HighestScore < 0 {
			return fmt.Errorf("stats file has negative values for %s", name)
		}
		if entry.Wins+entry.Draws > entry.Played {
			return fmt.Errorf("stats file has invalid results for %s", name)
		}
	}
	return nil
}

func highestScore(data file) int {
	highest := 0
	for _, difficulty := range difficultyOrder {
		if score := data.Difficulties[difficultyString(difficulty)].HighestScore; score > highest {
			highest = score
		}
	}
	return highest
}

func totalPlayed(data file) int {
	total := 0
	for _, difficulty := range difficultyOrder {
		total += data.Difficulties[difficultyString(difficulty)].Played
	}
	return total
}

func newFile() file {
	data := file{
		Version:      currentVersion,
		Difficulties: make(map[string]DifficultyStats, len(difficultyOrder)),
	}
	for _, difficulty := range difficultyOrder {
		data.Difficulties[difficultyString(difficulty)] = DifficultyStats{}
	}
	return data
}

func statsPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "yahtzee", "stats.json"), nil
}

func difficultyName(difficulty game.Difficulty) (string, bool) {
	if int(difficulty) < 0 || int(difficulty) >= len(difficultyOrder) {
		return "", false
	}
	return difficultyString(difficulty), true
}

func difficultyString(difficulty game.Difficulty) string {
	switch difficulty {
	case game.Easy:
		return "easy"
	case game.Normal:
		return "normal"
	case game.Expert:
		return "expert"
	default:
		return "unknown"
	}
}

func difficultyLabel(difficulty game.Difficulty) string {
	switch difficulty {
	case game.Easy:
		return "Easy"
	case game.Normal:
		return "Normal"
	case game.Expert:
		return "Expert"
	default:
		return "Unknown"
	}
}
