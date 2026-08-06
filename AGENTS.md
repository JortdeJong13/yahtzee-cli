# Yahtzee CLI

## Project shape

This repository contains one terminal-only Yahtzee game: one human player versus an expected-value opponent. Keep the implementation small and direct. Use the standard library wherever possible; `golang.org/x/term` is the only runtime dependency because the CLI needs raw keyboard input.

The game rules live in `internal/game`. Terminal input, ANSI handling, rendering, and the game loop live in `internal/ui`. Do not add strategy interfaces, configuration systems, persistence, networking, or a UI framework until a concrete feature needs them.

## Working agreements

- The CLI is the product. Always try behavior through the actual executable after changing it.
- Prefer manual black-box checks of user-visible behavior over unit-test scaffolding.
- Keep game-state transitions explicit and keep terminal-only concerns out of the game package.
- Handle errors at real boundaries: terminal setup, input/output, and process startup. Do not add defensive error plumbing between controlled internal components.
- Keep `docs/design.md` and `README.md` aligned with user-visible behavior.
- Run `gofmt` on changed Go files and run the verification commands below before handing work back.

## Verification

```sh
gofmt -w cmd/yahtzee/main.go internal/game/*.go internal/ui/*.go
go vet ./...
go build ./...
```

The interactive CLI should be exercised manually in a real terminal.
