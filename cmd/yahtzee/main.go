package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/JortdeJong13/yahtzee-cli/internal/game"
	"github.com/JortdeJong13/yahtzee-cli/internal/ui"
)

var version = "devel"

func main() {
	difficultyName := flag.String("difficulty", "normal", "opponent difficulty: easy, normal, or expert")
	showVersion := flag.Bool("version", false, "print version and exit")
	showShortVersion := flag.Bool("v", false, "print version and exit")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "yahtzee: unexpected arguments")
		os.Exit(2)
	}
	if *showVersion || *showShortVersion {
		fmt.Println("yahtzee", currentVersion())
		return
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

func currentVersion() string {
	if version != "devel" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}
