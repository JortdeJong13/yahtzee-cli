# Yahtzee CLI

## Scope and principles

- This is a small, terminal-only Yahtzee game: one human versus an expected-value opponent.
- Keep it direct and minimal. Do not add future-proofing, frameworks, configuration, persistence, networking, or strategy interfaces before a concrete feature needs them.
- Use Go and standard-library code wherever possible. `golang.org/x/term` is the only runtime dependency.
- Do not use Python for project code or tooling.
- The CLI is the product. Prefer manual black-box checks over unit-test scaffolding; add E2E coverage later when it provides clear value.

## Architecture

- `internal/game`: dice, scorecards, rules, turns, and opponent decisions.
- `internal/ui`: terminal input, ANSI handling, rendering, animation, and the game loop.
- Keep game-state transitions explicit and keep terminal concerns out of `internal/game`.
- Handle errors at real boundaries: terminal setup, input/output, and process startup. Avoid defensive error plumbing between controlled internal components.

## Important behavior

- The player starts. The initial five dice are displayed but do not count as a roll.
- Each turn allows up to three rolls. Dice values carry into the next turn; locks and roll state reset.
- The opponent has no score pointer. Every level chooses deterministically apart from dice rolls. Easy looks one reroll ahead and values all points earned now, including bonuses. Normal and Expert use full-turn expectimax and precomputed upper/lower scorecard continuations, with future weights of 0.4 and 1 respectively. `--difficulty normal` is the default. Difficulty must not introduce random mistakes or change scoring rules.
- Use classic Yahtzee rules: 13 categories, upper bonus at 63, Yahtzee for 50, and 100 points for each later Yahtzee after the Yahtzee box contains 50.
- Later Yahtzees use the Joker rules. If the matching upper category is open, it is required; otherwise use an open lower category. If all lower categories are filled, an open upper category scores zero. The same placement rules apply after a Yahtzee box scored zero, without the bonus.
- The live board uses the ANSI alternate screen. Completed games persist in terminal history; in-progress quits do not. Quit requires two `q` presses.
- Keep the current compact two-column layout and its established color behavior: available human scores are purple, available opponent scores are yellow, the pointer only adds bold, and confirmed categories/scores retain their player color.

## Working agreements

- Preserve unrelated user changes in the working tree.
- Always run the actual executable after user-visible changes.
- Run `gofmt` on changed Go files and the verification commands below before handing work back.
- Do not commit, tag, or push unless the user asks for it.

## Verification

```sh
gofmt -w cmd/yahtzee/main.go internal/game/*.go internal/ui/*.go
go vet ./...
go test ./...
go build ./...
```

Exercise the interactive CLI manually in a real terminal. If the environment cannot use its default Go caches, use writable temporary caches:

```sh
GOCACHE=/tmp/yahtzee-go-cache GOMODCACHE=/tmp/yahtzee-go-modcache go vet ./...
GOCACHE=/tmp/yahtzee-go-cache GOMODCACHE=/tmp/yahtzee-go-modcache go test ./...
GOCACHE=/tmp/yahtzee-go-cache GOMODCACHE=/tmp/yahtzee-go-modcache go build ./...
```

After opponent changes, run the opt-in score calibration on separate validation seeds:

```sh
go test -tags botcheck ./internal/game -run TestOpponentCalibration -count=1 -v
```

Use `-opponent-games`, `-opponent-seed`, and `-opponent-sweep` to adjust the sample and compare future weights. Decision and rule checks must pass before accepting a score target. Do not regenerate the continuation tables unless their model or format changes.

## Releases

- The canonical module and install path is `github.com/JortdeJong13/yahtzee-cli`.
- A `v*` tag triggers the GitHub Actions release workflow, which builds macOS, Linux, and Windows archives for amd64 and arm64 and publishes checksums.
- For a new release, after the changes are committed:

```sh
git push origin main
git tag vX.Y.Z
git push origin vX.Y.Z
```
