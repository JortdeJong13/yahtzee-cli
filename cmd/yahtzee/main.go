package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/JortdeJong13/yahtzee-cli/internal/game"
	"github.com/JortdeJong13/yahtzee-cli/internal/ui"
)

func main() {
	difficultyName := flag.String("difficulty", "normal", "opponent difficulty: easy, normal, or expert")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "yahtzee: unexpected arguments")
		os.Exit(2)
	}

	difficulty, err := game.ParseDifficulty(*difficultyName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "yahtzee:", err)
		os.Exit(2)
	}

	if err := ui.Run(os.Stdin, os.Stdout, difficulty); err != nil {
		fmt.Fprintln(os.Stderr, "yahtzee:", err)
		os.Exit(1)
	}
}
