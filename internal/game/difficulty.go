package game

import "fmt"

type Difficulty uint8

const (
	Easy Difficulty = iota
	Normal
	Expert
)

func ParseDifficulty(value string) (Difficulty, error) {
	switch value {
	case "easy":
		return Easy, nil
	case "normal":
		return Normal, nil
	case "expert":
		return Expert, nil
	default:
		return Normal, fmt.Errorf("invalid difficulty %q (select easy, normal, or expert)", value)
	}
}
