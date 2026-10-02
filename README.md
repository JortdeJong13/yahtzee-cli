# yahtzee-cli

A small terminal-only Yahtzee game for one player against an expected-value opponent.

## Run from source

Requires Go 1.26 or newer:

```sh
go run ./cmd/yahtzee
```

## Install

```sh
go install github.com/JortdeJong13/yahtzee-cli/cmd/yahtzee@latest
"$(go env GOPATH)/bin/yahtzee"
```

Go installs the binary in `$(go env GOPATH)/bin` by default. Add that directory to `PATH` if you want to run it as `yahtzee`.

Standalone binaries are also available from the [GitHub Releases](https://github.com/JortdeJong13/yahtzee-cli/releases) page.

Upgrade an installation made with `go install` by running the install command again:

```sh
go install github.com/JortdeJong13/yahtzee-cli/cmd/yahtzee@latest
yahtzee --version
```

The opponent uses normal difficulty by default. Choose another level with:

```sh
yahtzee --difficulty easy
yahtzee --difficulty expert
```

Every difficulty chooses the best move it can see, with no deliberate random mistakes. Easy looks one reroll ahead and values points earned now, including bonuses. Normal evaluates the full turn and gives future scorecard opportunities a reduced weight. Expert evaluates the full turn with the full future-score estimate; that estimate is approximate rather than a complete solution of the game.

For the opponent design, decision checks, and score calibration method, see [the opponent design notes](docs/opponent-design.md).

The difficulty flag also has a short form:

```sh
yahtzee -d expert
```

View completed-game statistics with:

```sh
yahtzee --stats
yahtzee -s
```

Use `yahtzee --help` for all available options.

## How to play

```text
                                     ┌────────────────────────────────────────┐
   ╔═══════╗           ╔═══════╗     │Category              You    Opponent   │
   ║       ║           ║       ║     │────────────────────────────────────────│
   ║   ●   ║           ║   ●   ║     │Ones                    3               │
   ║       ║           ║       ║     │Twos                    0               │
   ╚═══════╝           ╚═══════╝     │Threes                  0               │
      [1]                 [2]        │Fours                   8               │
             ╔═══════╗               │Fives                  15               │
             ║ ●   ● ║               │Sixes                   0               │
             ║       ║               │────────────────────────────────────────│
             ║ ●   ● ║               │Bonus               15/63        0/63   │
             ╚═══════╝               │────────────────────────────────────────│
                [3]                  │Three of a Kind        11               │
   ╔═══════╗           ╔═══════╗     │Four of a Kind          0          18   │
   ║       ║           ║ ●   ● ║     │Full House           → 25               │
   ║   ●   ║           ║       ║     │Small Straight          0               │
   ║       ║           ║ ●   ● ║     │Large Straight          0               │
   ╚═══════╝           ╚═══════╝     │Yahtzee                 0               │
      [4]                 [5]        │Chance                 11               │
                                     │────────────────────────────────────────│
   ▶ Your turn: select a score       │TOTAL                  15          18   │
                                     └────────────────────────────────────────┘
   [r] roll       [1-5] lock/unlock      [↑ ↓] select score      [↵] confirm
```

- `r`: roll
- `1`–`5`: lock or unlock a die
- Up/down arrows: select an open score category
- Enter: score the selected category
- `q`, then `q` again: quit
