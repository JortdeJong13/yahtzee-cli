package ui

import (
	"io"
)

type KeyKind uint8

const (
	KeyUnknown KeyKind = iota
	KeyRune
	KeyUp
	KeyDown
	KeyEnter
	KeyEscape
)

type Key struct {
	Kind KeyKind
	Rune rune
}

type KeyReader struct {
	reader io.Reader
}

func NewKeyReader(reader io.Reader) *KeyReader {
	return &KeyReader{reader: reader}
}

func (r *KeyReader) Read() (Key, error) {
	var first [1]byte
	if _, err := io.ReadFull(r.reader, first[:]); err != nil {
		return Key{}, err
	}

	switch first[0] {
	case '\r', '\n':
		return Key{Kind: KeyEnter}, nil
	case 0x1b:
		var sequence [2]byte
		if _, err := io.ReadFull(r.reader, sequence[:]); err != nil {
			return Key{Kind: KeyEscape}, err
		}
		if sequence[0] == '[' {
			switch sequence[1] {
			case 'A':
				return Key{Kind: KeyUp}, nil
			case 'B':
				return Key{Kind: KeyDown}, nil
			}
		}
		return Key{Kind: KeyEscape}, nil
	default:
		return Key{Kind: KeyRune, Rune: rune(first[0])}, nil
	}
}
