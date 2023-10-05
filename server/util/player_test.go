package util

import "testing"

func TestPlayer(t *testing.T) {
	deck := NewDeck()
	t.Log(deck.String())
	deck.Shuffle()
	// p1 := NewPlayer("yff")
	// p1.Deal(&deck, 17)
	// p1.Sort()
	// t.Log(p1.String())
}
