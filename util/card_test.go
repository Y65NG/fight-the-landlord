package util

import (
	"strings"
	"testing"
)

func TestCard(t *testing.T) {
	cards1 := []*Card {&Card{Point: THREE}, &Card{Point: FOUR}, &Card{Point: THREE}, &Card{Point: FOUR}, &Card{Point: FOUR}}
	Sort(cards1)
	t.Log(cards1)
	cards2 := []*Card {&Card{Point: TWO}, &Card{Point: THREE}, &Card{Point: THREE}, &Card{Point: THREE}, &Card{Point: TWO}}
	Sort(cards2)
	t.Log(cards2)
	t.Log(Valid(cards2))
	t.Log(CompareTo(cards1, cards2))
}

func TestSplit(t *testing.T) {
	str := "a, 2, 3"
	splittedStr := strings.Split(str, ", ")
	t.Log(splittedStr)
}
