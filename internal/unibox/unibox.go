package unibox

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Marker interface {
	MarkUnderflow() rune
	MarkLeft() rune
	Mark() rune
	Line() rune
	MarkRight() rune
	MarkOverflow() rune
}

type runeMarker struct {
	mu, ml, m, l, mr, mo rune
}

func (r *runeMarker) MarkUnderflow() rune { return r.mu }
func (r *runeMarker) MarkLeft() rune      { return r.ml }
func (r *runeMarker) Mark() rune          { return r.m }
func (r *runeMarker) Line() rune          { return r.l }
func (r *runeMarker) MarkRight() rune     { return r.mr }
func (r *runeMarker) MarkOverflow() rune  { return r.mo }

var (
	EdgeAboveMarker = &runeMarker{'╓', '┌', '┬', '─', '┐', '╖'}
	EdgeBelowMarker = &runeMarker{'╙', '└', '┴', '─', '┘', '╜'}
	LineBelowMarker = &runeMarker{'┹', '┵', '┴', '─', '┶', '┺'}
)

func MarkExp(src string, exp string, at int) string {
	msg := fmt.Sprintf("%v\n", exp)
	msg += fmt.Sprintf("%v [%3.1d]\n", src, len(src))
	msg += fmt.Sprintf(
		"%v [%3.1d]\n", MarkEdgeBelow(src, at), at)
	return msg
}

func MarkEdgeAbove(pat string, off int) string {
	return MarkWith(pat, off, EdgeAboveMarker)
}

func MarkEdgeBelow(pat string, off int) string {
	return MarkWith(pat, off, EdgeBelowMarker)
}

func MarkLineBelow(pat string, off int) string {
	return MarkWith(pat, off, LineBelowMarker)
}

func MarkRunes(pat string, off int, mu, ml, m, l, mr, mo rune) string {
	rm := runeMarker{mu, ml, m, l, mr, mo}
	return MarkWith(pat, off, &rm)
}

func MarkWith(pat string, off int, m Marker) string {
	if pat == `` {
		return ``
	}
	l := m.Line()
	if off < 0 {
		return string(m.MarkUnderflow()) +
			strings.Repeat(string(l), utf8.RuneCountInString(pat)-1)
	}
	if off >= len(pat) {
		return strings.Repeat(
			string(l), utf8.RuneCountInString(pat)-1) + string(m.MarkOverflow())
	}

	var buf bytes.Buffer
	for idx, r := range pat {
		box := l
		switch {
		case idx == 0 && off < utf8.RuneLen(r):
			box, off = m.MarkLeft(), len(pat)*2
		case utf8.RuneLen(r)+idx > off:
			if off == len(pat) {
				box, off = m.MarkRight(), len(pat)*2
			} else {
				box, off = m.Mark(), len(pat)*2
			}
		}
		buf.WriteRune(box)
	}
	return buf.String()
}
