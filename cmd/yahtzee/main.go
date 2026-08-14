package main

import (
	"fmt"
	"os"

	"github.com/JortdeJong13/yahtzee-cli/internal/ui"
)

func main() {
	if err := ui.Run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "yahtzee:", err)
		os.Exit(1)
	}
}
