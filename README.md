# yahtzee-cli

A small terminal-only Yahtzee game for one player against an expected-value opponent.

## Run from source

Requires Go 1.26 or newer:

```sh
go run ./cmd/yahtzee
```

## Install

```sh
go install github.com/jortdejong/yahtzee-cli/cmd/yahtzee@latest
yahtzee
```

GitHub Releases will provide standalone binaries once the first release is published.

To publish a release after the repository is on GitHub:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds macOS, Linux, and Windows binaries for amd64 and arm64, then attaches archives and checksums to the GitHub release.

## Controls

- `r`: roll
- `1`–`5`: lock or unlock a die
- Up/down arrows: select an open score category
- Enter: score the selected category
- `q`, then `q` again: quit

Set `NO_COLOR=1` if ANSI colors are not wanted.

## Development checks

```sh
gofmt -w cmd/yahtzee/main.go internal/game/*.go internal/ui/*.go
go vet ./...
go build ./...
```
