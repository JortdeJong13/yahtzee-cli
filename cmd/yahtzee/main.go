package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/JortdeJong13/yahtzee-cli/internal/game"
	"github.com/JortdeJong13/yahtzee-cli/internal/stats"
	"github.com/JortdeJong13/yahtzee-cli/internal/ui"
	"golang.org/x/term"
)

var version = "devel"

func main() {
	var difficultyName string
	var showHelp, showStats, showVersion bool
	flag.StringVar(&difficultyName, "difficulty", "normal", "opponent difficulty: easy, normal, or expert")
	flag.StringVar(&difficultyName, "d", "normal", "shorthand for --difficulty")
	flag.BoolVar(&showStats, "stats", false, "print game statistics and exit")
	flag.BoolVar(&showStats, "s", false, "shorthand for --stats")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&showVersion, "v", false, "shorthand for --version")
	flag.BoolVar(&showHelp, "help", false, "print this help message")
	flag.BoolVar(&showHelp, "h", false, "shorthand for --help")
	flag.Usage = func() {
		printUsage(os.Stderr)
	}
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "yahtzee: unexpected arguments")
		os.Exit(2)
	}
	if showHelp {
		printUsage(os.Stdout)
		return
	}
	if showVersion {
		fmt.Println("yahtzee", currentVersion())
		return
	}
	if showStats {
		colors := os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(os.Stdout.Fd()))
		if err := stats.Print(os.Stdout, colors); err != nil {
			fmt.Fprintln(os.Stderr, "yahtzee:", err)
			os.Exit(1)
		}
		return
	}

	difficulty, err := game.ParseDifficulty(difficultyName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "yahtzee:", err)
		os.Exit(2)
	}

	if err := ui.Run(os.Stdin, os.Stdout, difficulty); err != nil {
		fmt.Fprintln(os.Stderr, "yahtzee:", err)
		os.Exit(1)
	}
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, "Usage: yahtzee [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -d, --difficulty easy|normal|expert")
	fmt.Fprintln(w, "      choose opponent difficulty (default: normal)")
	fmt.Fprintln(w, "  -s, --stats")
	fmt.Fprintln(w, "      print game statistics and exit")
	fmt.Fprintln(w, "  -v, --version")
	fmt.Fprintln(w, "      print version and exit")
	fmt.Fprintln(w, "  -h, --help")
	fmt.Fprintln(w, "      print this help message")
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
