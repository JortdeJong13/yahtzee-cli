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

The opponent uses normal difficulty by default. Choose another level with:

```sh
yahtzee --difficulty easy
yahtzee --difficulty expert
```

Easy and normal occasionally choose a near-best move. Expert always chooses the highest-value move and is deterministic apart from the dice rolls.

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
